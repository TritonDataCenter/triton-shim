package utils

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"regexp"
	"strings"
	"time"

	awscredentials "github.com/aws/aws-sdk-go/aws/credentials"
	awsv4signer "github.com/aws/aws-sdk-go/aws/signer/v4"
	"github.com/gin-gonic/gin"
	tritonaccount "github.com/joyent/triton-go/v2/account"
	tritonutils "github.com/joyent/triton-shim/utils/triton"
	"github.com/rs/zerolog/log"
)

const iSO8601BasicFormat = "20060102T150405Z"

// gin-gonic/gin#1295 since we cannot use `ShouldBindBodyWith`
func getRawBody(c *gin.Context) io.ReadSeeker {
	var signBody io.ReadSeeker
	if strings.ToLower(c.Request.Method) == "post" {
		buf := make([]byte, 1024)
		num, _ := c.Request.Body.Read(buf)
		reqBody := string(buf[0:num])
		c.Request.Body = ioutil.NopCloser(bytes.NewBuffer([]byte(reqBody))) // Write body back
		// Need a Seeker for the signer.Sign function. Let's build one
		// from the rawBodyString:
		signBody = bytes.NewReader([]byte(reqBody))
	}
	return signBody
}

func getAccessKeys(c *gin.Context) ([]*tritonaccount.AccessKey, error) {
	// TODO: Need to get account details from the request so we can
	// fetch access keys for the provided account, and not the one
	// we're using to operate as Admin
	client, err := tritonutils.GetTritonAccountClient()
	if err != nil {
		return nil, fmt.Errorf("Unable to create triton account client: %w", err)
	}
	ctx := context.Background()
	input := &tritonaccount.ListAccessKeysInput{}
	accesskeys, err := client.AccessKeys().ListAccessKeys(ctx, input)
	if err != nil {
		log.Printf("[ERROR] list accesskeys error: %v\n", err)
		return nil, fmt.Errorf("Unable to retrieve triton account keys: %w", err)
	}
	return accesskeys, nil
}

// The provided AccessKeyID should be part of the given Auhorization header,
// exactly right at the begining of the "Credential=" part of the header like:
// Credential=620f1a7322b6a26c9301c6bcc17ccff3/...
// We'll get it from the header, look for the key into access keys and, if we
// find it, we'll use to reconstruct the Authorization header for the current
// request, in order to verify if the provided signature matches.
func getCurrentKey(authHeader string, accesskeys []*tritonaccount.AccessKey) (*tritonaccount.AccessKey, error) {
	re := regexp.MustCompile(`Credential=(\w+)`)
	if !re.MatchString(authHeader) {
		return nil, errors.New("Unable to find the AccessKeyID used to sign the request")

	}

	credentials := re.FindStringSubmatch(authHeader)

	var currentKey *tritonaccount.AccessKey

	for _, aKey := range accesskeys {
		if aKey.AccessKeyID == credentials[1] {
			currentKey = aKey
			break
		}
	}

	if currentKey == nil {
		return nil, errors.New("The provided AccessKey does not belong to this account")
	}

	return currentKey, nil
}

func getSignedHeaders(authHeader string) ([]string, error) {
	// Once we know which AccessKey has been used to sign the request
	// we need to know which headers have been signed
	headersRe := regexp.MustCompile(`SignedHeaders=(\S+),`)
	if !headersRe.MatchString(authHeader) {
		return nil, errors.New("Unable to find signed headers")
	}
	signedHeaders := headersRe.FindStringSubmatch(authHeader)
	signedHeaders = strings.Split(signedHeaders[1], ";")
	return signedHeaders, nil
}

func compareSignatures(providedSignature string, calculatedSignature string) bool {
	signatureRe := regexp.MustCompile(`Signature=(\w+)`)
	if !signatureRe.MatchString(providedSignature) {
		return false
	}
	provided := signatureRe.FindStringSubmatch(providedSignature)

	if !signatureRe.MatchString(calculatedSignature) {
		return false
	}
	calculated := signatureRe.FindStringSubmatch(calculatedSignature)

	return provided[1] == calculated[1]
}

// VerifySignature middleware preloads access keys for the provided account,
// verifies that one of those keys is used to sign the HTTP Request and the
// correctness of the provided Signature
func VerifySignature() gin.HandlerFunc {
	return func(c *gin.Context) {
		// We'll use a copy of the current request to re-sign it and
		// verify if the provided signature matches with the one we
		// calculate
		dupeRequest := c.Request.Clone(c.Request.Context())

		// Load all the access keys:
		accesskeys, err := getAccessKeys(c)
		if err != nil {
			c.AbortWithError(http.StatusInternalServerError, err)
			return
		}
		// Get the Authorization header:
		authHeader := c.Request.Header.Get("Authorization")
		log.Printf("[Authorization Header]: %s\n", authHeader)

		if authHeader == "" {
			c.AbortWithError(http.StatusUnauthorized,
				errors.New("No Authentication header provided"))
			return
		}
		// Infer AccessKey in use from the auth header
		currentKey, err := getCurrentKey(authHeader, accesskeys)
		if err != nil {
			c.AbortWithError(http.StatusUnauthorized, err)
			return
		}

		signedHeaders, err := getSignedHeaders(authHeader)
		if err != nil {
			c.AbortWithError(http.StatusUnauthorized, err)
			return
		}

		// The time used to sign the request can be obtained from x-amz-date header:
		t, err := time.Parse(iSO8601BasicFormat, c.GetHeader("x-amz-date"))
		if err != nil {
			c.AbortWithError(http.StatusUnauthorized,
				errors.New("Unable to find request time"))
			return
		}

		dupeRequest.Header.Del("Authorization")
		// Only keep headers present in signedHeaders in the request we are about to
		// sign so it is done exactly with the same headers than the original one:
		for hdr := range dupeRequest.Header {
			hdrName := strings.ToLower(hdr)
			found := func(find string) bool {
				for _, val := range signedHeaders {
					if val == find {
						return true
					}
				}
				return false
			}(hdrName)

			if !found {
				dupeRequest.Header.Del(hdr)
			}
		}

		creds := awscredentials.NewStaticCredentials(currentKey.AccessKeyID, currentKey.SecretAccessKey, "")
		signer := awsv4signer.NewSigner(creds)
		signBody := getRawBody(c)

		_, err = signer.Sign(dupeRequest, signBody, "ec2", c.Request.Host, t)
		if err != nil {
			c.AbortWithError(http.StatusUnauthorized,
				errors.New("Unable to verify request signature"))
			return
		}
		log.Printf("[Calculated Authorization Header]: %s\n", dupeRequest.Header.Get("Authorization"))

		if !compareSignatures(authHeader, dupeRequest.Header.Get("Authorization")) {
			c.AbortWithError(http.StatusUnauthorized,
				errors.New("The provided request signature is not correct"))
			return
		}

		// TODO: The same thing but for QueryString Auth instead of Authentication Header

		c.Next()
	}
}
