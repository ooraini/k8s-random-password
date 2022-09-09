package main

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/strategicpatch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"log"
	"math/big"
	"os"
	"time"
)

func init() {
	assertAvailablePRNG()
}

func assertAvailablePRNG() {
	// Assert that a cryptographically secure PRNG is available.
	// Panic otherwise.
	buf := make([]byte, 1)

	_, err := io.ReadFull(rand.Reader, buf)
	if err != nil {
		panic(fmt.Sprintf("crypto/rand is unavailable: Read() failed with %#v", err))
	}
}

// GenerateRandomBytes returns securely generated random bytes.
// It will return an error if the system's secure random
// number generator fails to function correctly, in which
// case the caller should not continue.
func GenerateRandomBytes(n int) ([]byte, error) {
	b := make([]byte, n)
	_, err := rand.Read(b)
	// Note that err == nil only if we read len(b) bytes.
	if err != nil {
		return nil, err
	}

	return b, nil
}

// GenerateRandomString returns a securely generated random string.
// It will return an error if the system's secure random
// number generator fails to function correctly, in which
// case the caller should not continue.
func GenerateRandomString(n int) (string, error) {
	const letters = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz-"
	ret := make([]byte, n)
	for i := 0; i < n; i++ {
		num, err := rand.Int(rand.Reader, big.NewInt(int64(len(letters))))
		if err != nil {
			return "", err
		}
		ret[i] = letters[num.Int64()]
	}

	return string(ret), nil
}

func main() {
	// creates the in-cluster config
	config, err := rest.InClusterConfig()
	if err != nil {
		panic(err.Error())
	}
	// creates the clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}

	namespace, exists := os.LookupEnv("NAMESPACE")
	if !exists {
		log.Fatal("NAMESPACE environment variable is not set")
	}

	name, exists := os.LookupEnv("SECRET_NAME")
	if !exists {
		log.Fatal("SECRET_NAME environment variable is not set")
	}

	key, exists := os.LookupEnv("SECRET_KEY")
	if !exists {
		log.Fatal("SECRET_KEY environment variable is not set")
	}

	log.Printf("Namespace: '%s' Name: '%s' Key: %s", namespace, name, key)

	failures := 0
	waited := 0

	annotation := "k8s-random-password-generation-time"

	for {
		if failures >= 5 {
			log.Print("Unable to update secret")
			os.Exit(1)
		}

		secret, err := clientset.CoreV1().Secrets(namespace).Get(context.Background(), name, metav1.GetOptions{})
		if err != nil {
			if !errors.IsNotFound(err) {
				log.Print(err)
				failures = failures + 1
				time.Sleep(time.Second * time.Duration(failures) * 2)
				continue
			}

			if waited >= 30 {
				log.Print("secret not found, exiting")
				os.Exit(1)
			}

			log.Printf("secret not found, waiting")
			time.Sleep(time.Second * 10)
			waited = waited + 1
			continue
		}

		newSecret := secret.DeepCopy()

		if newSecret.Annotations == nil {
			newSecret.Annotations = map[string]string{}
		}

		_, exists := newSecret.Annotations[annotation]
		if exists {
			log.Printf("Secret contains annotation '%s', exiting", annotation)
			break
		}

		randomString, err := GenerateRandomString(31)
		if err != nil {
			log.Print(err)
			failures = failures + 1
			time.Sleep(time.Second * time.Duration(failures) * 2)
			continue
		}

		newSecret.Annotations[annotation] = time.Now().String()

		if newSecret.StringData == nil {
			newSecret.StringData = map[string]string{}
		}

		newSecret.StringData[key] = randomString

		secretJson, err := json.Marshal(secret)
		if err != nil {
			log.Print(err)
			failures = failures + 1
			time.Sleep(time.Second * time.Duration(failures) * 2)
			continue
		}

		newSecretJson, err := json.Marshal(newSecret)
		if err != nil {
			log.Print(err)
			failures = failures + 1
			time.Sleep(time.Second * time.Duration(failures) * 2)
			continue
		}

		patch, err := strategicpatch.CreateTwoWayMergePatch(secretJson, newSecretJson, v1.Secret{})
		if err != nil {
			log.Print(err)
			failures = failures + 1
			time.Sleep(time.Second * time.Duration(failures) * 2)
			continue
		}

		_, err = clientset.CoreV1().Secrets(namespace).Patch(context.Background(), name, types.StrategicMergePatchType, patch, metav1.PatchOptions{})
		if err != nil {
			log.Print(err)
			failures = failures + 1
			time.Sleep(time.Second * time.Duration(failures) * 2)
			continue
		}

		log.Print("Secret patched")

		break
	}
}
