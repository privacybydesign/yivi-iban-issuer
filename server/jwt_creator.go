package main

import (
	"crypto/rsa"
	"os"

	"github.com/golang-jwt/jwt/v4"
	"github.com/privacybydesign/irmago/irma"
)

type JwtCreator interface {
	CreateJwt(fullname string, iban string, bic string) (jwt string, err error)
}

func NewIrmaJwtCreator(privateKeyPath string,
	issuerId string,
	crediential string,
	sdJwtBatchSize uint,
) (*DefaultJwtCreator, error) {
	keyBytes, err := os.ReadFile(privateKeyPath)

	if err != nil {
		return nil, err
	}

	privateKey, err := jwt.ParseRSAPrivateKeyFromPEM(keyBytes)

	if err != nil {
		return nil, err
	}

	return &DefaultJwtCreator{
		issuerId:       issuerId,
		privateKey:     privateKey,
		credential:     crediential,
		sdJwtBatchSize: sdJwtBatchSize,
	}, nil
}

type DefaultJwtCreator struct {
	privateKey     *rsa.PrivateKey
	issuerId       string
	credential     string
	sdJwtBatchSize uint
}

func (jc *DefaultJwtCreator) CreateJwt(fullname string, iban string, bic string) (string, error) {
	issuanceRequest := irma.NewIssuanceRequest([]*irma.CredentialRequest{
		{
			CredentialTypeID: irma.NewCredentialTypeIdentifier(jc.credential),
			Attributes: map[string]string{
				"fullname": fullname,
				"iban":     iban,
				"bic":      bic,
			},
			SdJwtBatchSize: jc.sdJwtBatchSize,
		},
	})

	return irma.SignSessionRequest(
		issuanceRequest,
		jwt.GetSigningMethod(jwt.SigningMethodRS256.Alg()),
		jc.privateKey,
		jc.issuerId,
	)
}
