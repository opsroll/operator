/*
Copyright AppsCode Inc. and Contributors

Licensed under the AppsCode Community License 1.0.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    https://github.com/appscode/licenses/raw/1.0.0/AppsCode-Community-1.0.0.md

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package search_guard

import (
	"context"
	"fmt"

	api "kubedb.dev/apimachinery/apis/kubedb/v1alpha2"
	certlib "kubedb.dev/elasticsearch/pkg/lib/cert"
	"kubedb.dev/elasticsearch/pkg/lib/user"

	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"
	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	api_util "kmodules.xyz/client-go/api/v1"
	core_util "kmodules.xyz/client-go/core/v1"
)

const (
	ConfigFileName       = "elasticsearch.yml"
	InternalUserFileName = "sg_internal_users.yml"
	RolesMappingFileName = "sg_roles_mapping.yml"
)

var adminDNTemplate = `
searchguard.authcz.admin_dn:
- "%s"
`

var nodesDNTemplate = `
searchguard.nodes_dn:
- "%s"
`

var searchguard_security_enabled = `
searchguard.enterprise_modules_enabled: false

searchguard.ssl.transport.enforce_hostname_verification: false
searchguard.ssl.transport.pemkey_filepath: certs/transport/tls.key
searchguard.ssl.transport.pemcert_filepath: certs/transport/tls.crt
searchguard.ssl.transport.pemtrustedcas_filepath: certs/transport/ca.crt

searchguard.allow_unsafe_democertificates: true
searchguard.allow_default_init_sgindex: true
searchguard.enable_snapshot_restore_privilege: true
searchguard.check_snapshot_restore_write_privileges: true
searchguard.audit.type: internal_elasticsearch

searchguard.restapi.roles_enabled: ["SGS_ALL_ACCESS","sg_all_access"]
`

var searchguard_security_disabled = `
searchguard.disabled: true
`

var https_enabled = `
searchguard.ssl.http.enabled: true
searchguard.ssl.http.pemkey_filepath: certs/http/tls.key
searchguard.ssl.http.pemcert_filepath: certs/http/tls.crt
searchguard.ssl.http.pemtrustedcas_filepath: certs/http/ca.crt

# searchguard.authcz.admin_dn:
%s

# searchguard.nodes_dn:
%s
`

var https_disabled = `
searchguard.ssl.http.enabled: false
`

func (es *Elasticsearch) EnsureDefaultConfig() error {
	secret, err := es.findSecret(es.elasticsearch.ConfigSecretName())
	if err != nil {
		return err
	}

	if secret != nil {
		// If the secret already exists,
		// check whether it contains "elasticsearch.yml" file or not.
		if value, ok := secret.Data[ConfigFileName]; !ok || len(value) == 0 {
			return errors.New("elasticsearch.yml is missing")
		}

		// If secret is owned by the elasticsearch object,
		// update the labels.
		// Labels hold information like elasticsearch version,
		// should be synced.
		ctrl := metav1.GetControllerOf(secret)
		if ctrl != nil &&
			ctrl.Kind == api.ResourceKindElasticsearch && ctrl.Name == es.elasticsearch.Name {

			// sync labels
			if _, _, err := core_util.CreateOrPatchSecret(context.TODO(), es.kClient, secret.ObjectMeta, func(in *core.Secret) *core.Secret {
				in.Labels = core_util.UpsertMap(in.Labels, es.elasticsearch.OffshootLabels())
				return in
			}, metav1.PatchOptions{}); err != nil {
				return err
			}
		}

		return nil
	}

	// config secret isn't created yet.
	// let's create it.
	owner := metav1.NewControllerRef(es.elasticsearch, api.SchemeGroupVersion.WithKind(api.ResourceKindElasticsearch))
	secretMeta := metav1.ObjectMeta{
		Name:      es.elasticsearch.ConfigSecretName(),
		Namespace: es.elasticsearch.Namespace,
	}

	var config, inUserConfig, rolesMapping string

	if !es.elasticsearch.Spec.DisableSecurity {
		config = searchguard_security_enabled

		// password for default users: admin, kibanaserver, etc.
		inUserConfig, err = es.getInternalUserConfig()
		if err != nil {
			return errors.Wrap(err, "failed to generate default internal user config")
		}

		rolesMapping, err = es.getRolesMapping()
		if err != nil {
			return errors.Wrap(err, "failed to generate default roles_mapping.yml")
		}

		// If rest layer is secured with certs
		if es.elasticsearch.Spec.EnableSSL {
			if es.elasticsearch.Spec.TLS == nil {
				return errors.New("spec.TLS configuration is empty")
			}

			// Get transport layer cert secret.
			// Parse the tls.cert to extract the nodeDNs.
			sName, exist := api_util.GetCertificateSecretName(es.elasticsearch.Spec.TLS.Certificates, string(api.ElasticsearchTransportCert))
			if !exist {
				return errors.New("transport-cert secret is missing")
			}

			cSecret, err := es.kClient.CoreV1().Secrets(es.elasticsearch.Namespace).Get(context.TODO(), sName, metav1.GetOptions{})
			if err != nil {
				return errors.Wrap(err, fmt.Sprintf("failed to get certificateSecret: %s/%s", es.elasticsearch.Namespace, sName))
			}

			nodesDN := ""
			if value, ok := cSecret.Data[certlib.TLSCert]; ok {
				subj, err := certlib.ExtractSubjectFromCertificate(value)
				if err != nil {
					return err
				}
				nodesDN = fmt.Sprintf(nodesDNTemplate, subj.String())
			}

			// Get searchguard admin cert secret.
			// Parse the tls.cert to extract the adminDNs.
			sName, exist = api_util.GetCertificateSecretName(es.elasticsearch.Spec.TLS.Certificates, string(api.ElasticsearchAdminCert))
			if !exist {
				return errors.New("admin-cert secret is missing")
			}

			cSecret, err = es.kClient.CoreV1().Secrets(es.elasticsearch.Namespace).Get(context.TODO(), sName, metav1.GetOptions{})
			if err != nil {
				return errors.Wrap(err, fmt.Sprintf("failed to get certificateSecret: %s/%s", es.elasticsearch.Namespace, sName))
			}

			adminDN := ""
			if value, ok := cSecret.Data[certlib.TLSCert]; ok {
				subj, err := certlib.ExtractSubjectFromCertificate(value)
				if err != nil {
					return err
				}
				adminDN = fmt.Sprintf(adminDNTemplate, subj.String())
			}

			config += fmt.Sprintf(https_enabled, adminDN, nodesDN)
		} else {
			config += https_disabled
		}

	} else {
		config = searchguard_security_disabled
	}

	data := map[string][]byte{
		ConfigFileName:       []byte(config),
		InternalUserFileName: []byte(inUserConfig),
		RolesMappingFileName: []byte(rolesMapping),
	}

	if _, _, err := core_util.CreateOrPatchSecret(context.TODO(), es.kClient, secretMeta, func(in *core.Secret) *core.Secret {
		in.Labels = core_util.UpsertMap(in.Labels, es.elasticsearch.OffshootLabels())
		core_util.EnsureOwnerReference(&in.ObjectMeta, owner)
		in.Data = data
		return in
	}, metav1.PatchOptions{}); err != nil {
		return err
	}

	return nil
}

func (es *Elasticsearch) getInternalUserConfig() (string, error) {
	userList := es.elasticsearch.Spec.InternalUsers
	if userList == nil {
		return "", errors.New("spec.internalUsers is empty")
	}

	for username := range userList {
		var pass string
		var err error

		if username == string(api.ElasticsearchInternalUserAdmin) {
			pass, err = es.getPasswordFromSecret(es.elasticsearch.Spec.AuthSecret.Name)
			if err != nil {
				return "", errors.Wrap(err, fmt.Sprintf("failed to get password from secret: %s/%s", es.elasticsearch.Namespace, es.elasticsearch.Spec.AuthSecret.Name))
			}
		} else {
			pass, err = es.getPasswordFromSecret(es.elasticsearch.UserCredSecretName(username))
			if err != nil {
				return "", errors.Wrap(err, fmt.Sprintf("failed to get password from secret: %s/%s", es.elasticsearch.Namespace, es.elasticsearch.UserCredSecretName(username)))
			}
		}

		err = user.SetPasswordHashForUser(userList, username, pass)
		if err != nil {
			return "", errors.Wrap(err, "failed to generate the password hash")
		}
	}

	byt, err := yaml.Marshal(userList)
	if err != nil {
		return "", errors.Wrap(err, "failed to marshal the internal user list")
	}

	return string(byt), nil
}

func (es *Elasticsearch) getRolesMapping() (string, error) {
	rolesMapping := es.elasticsearch.Spec.RolesMapping
	// if rolesMapping is nil, return empty string
	// no need to  perform yaml.Marshal().
	// coz it will generate ( `{}` ).
	if rolesMapping == nil {
		return "", nil
	}

	byt, err := yaml.Marshal(rolesMapping)
	if err != nil {
		return "", errors.Wrap(err, "failed to marshal the roles mapping")
	}

	return string(byt), nil
}

func (es *Elasticsearch) getPasswordFromSecret(sName string) (string, error) {
	secret, err := es.kClient.CoreV1().Secrets(es.elasticsearch.Namespace).Get(context.TODO(), sName, metav1.GetOptions{})
	if err != nil {
		return "", err
	}

	if value, exist := secret.Data[core.BasicAuthPasswordKey]; exist && len(value) != 0 {
		return string(value), nil
	}

	return "", errors.New(fmt.Sprintf("password is missing in secret: %s/%s", es.elasticsearch.Namespace, sName))
}
