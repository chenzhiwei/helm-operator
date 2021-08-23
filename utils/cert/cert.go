package cert

import (
	"bytes"
	"context"
	"fmt"
	"time"

	certctl "github.com/chenzhiwei/certctl/pkg/cert"
	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/chenzhiwei/helm-operator/utils/constant"
)

func SetupTLSCert(c client.Client, reader client.Reader) error {
	ctx := context.TODO()

	secret := &corev1.Secret{}
	if err := reader.Get(ctx, types.NamespacedName{Name: constant.HelmOperatorTLSSecretName, Namespace: constant.HelmOperatorNamespace}, secret); err != nil {
		if !errors.IsNotFound(err) {
			return err
		}

		secret, err = newSecret(ctx, c)
		if err != nil {
			return err
		}

		if err := c.Create(ctx, secret); err != nil {
			if !errors.IsAlreadyExists(err) {
				return err
			}
		}
	}

	webhookConfig := &admissionregistrationv1.ValidatingWebhookConfiguration{}
	if err := reader.Get(ctx, types.NamespacedName{Name: constant.HelmOperatorWebhookConfigName}, webhookConfig); err != nil {
		return err
	}

	for i, webhook := range webhookConfig.Webhooks {
		if webhook.Name == constant.HelmOperatorWebhookName {
			if bytes.Compare(webhook.ClientConfig.CABundle, secret.Data["tls.crt"]) == 0 {
				return nil
			} else {
				webhookConfig.Webhooks[i].ClientConfig.CABundle = secret.Data["tls.crt"]
				break
			}
		}
	}

	if err := c.Update(ctx, webhookConfig); err != nil {
		return err
	}

	return nil
}

func newSecret(ctx context.Context, c client.Client) (*corev1.Secret, error) {
	duration := time.Hour * 24 * 3650
	subject := "CN=helm-operator-webhook/O=helm-operator"
	san := fmt.Sprintf("%s.%s.svc", constant.HelmOperatorServiceName, constant.HelmOperatorNamespace)
	certInfo, err := certctl.NewCertInfo(duration, subject, san, "", "", true)
	if err != nil {
		return nil, err
	}

	certBytes, keyBytes, err := certctl.NewCACertKey(certInfo, 2048)
	if err != nil {
		return nil, err
	}

	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      constant.HelmOperatorTLSSecretName,
			Namespace: constant.HelmOperatorNamespace,
		},

		Type: corev1.SecretTypeTLS,

		Data: map[string][]byte{
			"tls.crt": certBytes,
			"tls.key": keyBytes,
		},
	}, nil
}
