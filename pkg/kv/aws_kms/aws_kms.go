package aws_kms

import (
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/kms"

	"gitlab.jetstack.net/jetstack-experimental/vault-unsealer/pkg/kv"
)

type awsKMS struct {
	store      kv.Service
	kmsService *kms.KMS

	kmsID string
}

var _ kv.Service = &awsKMS{}

func New(store kv.Service, kmsID string) (kv.Service, error) {
	sess, err := session.NewSession()
	if err != nil {
		return nil, err
	}

	return &awsKMS{
		store:      store,
		kmsService: kms.New(sess),
		kmsID:      kmsID,
	}, nil
}

func (a *awsKMS) decrypt(cipherText []byte) ([]byte, error) {
	out, err := a.kmsService.Decrypt(&kms.DecryptInput{
		CiphertextBlob: cipherText,
		EncryptionContext: map[string]*string{
			"Key": aws.String("EncryptionContextValue"), // Required
		},
		GrantTokens: []*string{
			aws.String("GrantTokenType"), // Required
		},
	})
	return out.Plaintext, err
}

func (a *awsKMS) Get(key string) ([]byte, error) {
	cipherText, err := a.store.Get(key)
	if err != nil {
		return nil, err
	}

	return a.decrypt(cipherText)
}

func (a *awsKMS) encrypt(plainText []byte) ([]byte, error) {

	out, err := a.kmsService.Encrypt(&kms.EncryptInput{
		KeyId:     aws.String(a.kmsID),
		Plaintext: plainText,
		EncryptionContext: map[string]*string{
			"Key": aws.String("EncryptionContextValue"),
		},
		GrantTokens: []*string{
			aws.String("GrantTokenType"),
		},
	})
	return out.CiphertextBlob, err
}

func (a *awsKMS) Set(key string, val []byte) error {
	cipherText, err := a.encrypt(val)

	if err != nil {
		return err
	}

	return a.store.Set(key, cipherText)
}

func (g *awsKMS) Test(key string) error {
	inputString := "test"

	cipherText, err := g.encrypt([]byte(inputString))
	if err != nil {
		return err
	}

	plainText, err := g.decrypt(cipherText)
	if err != nil {
		return err
	}

	if string(plainText) != inputString {
		return fmt.Errorf("encryped and decryped text doesn't match: exp: '%v', act: '%v'", inputString, string(plainText))
	}

	return nil
}
