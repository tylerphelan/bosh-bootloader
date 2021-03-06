package actors

import (
	"bytes"
	"errors"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/cloudfoundry/bosh-bootloader/integration-test"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
)

const (
	AWSIAAS = iota
	GCPIAAS
)

type BBL struct {
	stateDirectory string
	pathToBBL      string
	configuration  integration.Config
}

type IAAS int

func NewBBL(stateDirectory string, pathToBBL string, configuration integration.Config) BBL {
	return BBL{
		stateDirectory: stateDirectory,
		pathToBBL:      pathToBBL,
		configuration:  configuration,
	}
}

func (b BBL) Up(iaas IAAS) {
	args := []string{
		"--state-dir", b.stateDirectory,
		"up",
	}

	switch iaas {
	case AWSIAAS:
		args = append(args, []string{
			"--iaas", "aws",
			"--aws-access-key-id", b.configuration.AWSAccessKeyID,
			"--aws-secret-access-key", b.configuration.AWSSecretAccessKey,
			"--aws-region", b.configuration.AWSRegion,
		}...)
	case GCPIAAS:
		args = append(args, []string{
			"--iaas", "gcp",
			"--gcp-service-account-key", b.configuration.GCPServiceAccountKeyPath,
			"--gcp-project-id", b.configuration.GCPProjectID,
			"--gcp-region", b.configuration.GCPRegion,
			"--gcp-zone", b.configuration.GCPZone,
		}...)
	default:
		panic(errors.New("invalid iaas"))
	}

	session := b.execute(args, os.Stdout, os.Stderr)
	Eventually(session, 40*time.Minute).Should(gexec.Exit(0))
}

func (b BBL) UpWithInvalidAWSCredentials() {
	args := []string{
		"--state-dir", b.stateDirectory,
		"up",
		"--iaas", "aws",
		"--aws-access-key-id", "some-bad-access-key-id",
		"--aws-secret-access-key", "some-bad-secret-access-key",
		"--aws-region", b.configuration.AWSRegion,
	}
	session := b.execute(args, os.Stdout, os.Stderr)
	Eventually(session, 10*time.Second).Should(gexec.Exit(1))
}

func (b BBL) Destroy() {
	session := b.execute([]string{
		"--state-dir", b.stateDirectory,
		"destroy",
		"--no-confirm",
	}, os.Stdout, os.Stderr)
	Eventually(session, 10*time.Minute).Should(gexec.Exit(0))
}

func (b BBL) SaveDirectorCA() string {
	stdout := bytes.NewBuffer([]byte{})
	session := b.execute([]string{
		"--state-dir", b.stateDirectory,
		"director-ca-cert",
	}, stdout, os.Stderr)
	Eventually(session, 10*time.Minute).Should(gexec.Exit(0))

	file, err := ioutil.TempFile("", "")
	defer file.Close()
	Expect(err).NotTo(HaveOccurred())

	file.Write(stdout.Bytes())

	return file.Name()
}

func (b BBL) DirectorUsername() string {
	return b.fetchValue("director-username")
}

func (b BBL) DirectorPassword() string {
	return b.fetchValue("director-password")
}

func (b BBL) DirectorAddress() string {
	return b.fetchValue("director-address")
}

func (b BBL) EnvID() string {
	return b.fetchValue("env-id")
}

func (b BBL) CreateLB(loadBalancerType string, cert string, key string, chain string) {
	args := []string{
		"--state-dir", b.stateDirectory,
		"create-lbs",
		"--type", loadBalancerType,
		"--cert", cert,
		"--key", key,
		"--chain", chain,
	}

	session := b.execute(args, os.Stdout, os.Stderr)
	Eventually(session, 10*time.Minute).Should(gexec.Exit(0))
}

func (b BBL) CreateGCPLB(loadBalancerType string) {
	args := []string{
		"--state-dir", b.stateDirectory,
		"create-lbs",
		"--type", loadBalancerType,
	}

	session := b.execute(args, os.Stdout, os.Stderr)
	Eventually(session, 10*time.Minute).Should(gexec.Exit(0))
}

func (b BBL) LBs() *gexec.Session {
	args := []string{
		"--state-dir", b.stateDirectory,
		"lbs",
	}

	session := b.execute(args, os.Stdout, os.Stderr)
	Eventually(session, 10*time.Minute).Should(gexec.Exit(0))

	return session
}

func (b BBL) UpdateLB(certPath, keyPath string) {
	args := []string{
		"--state-dir", b.stateDirectory,
		"update-lbs",
		"--cert", certPath,
		"--key", keyPath,
	}

	session := b.execute(args, os.Stdout, os.Stderr)
	Eventually(session, 10*time.Minute).Should(gexec.Exit(0))
}

func (b BBL) DeleteLB() {
	args := []string{
		"--state-dir", b.stateDirectory,
		"delete-lbs",
	}

	session := b.execute(args, os.Stdout, os.Stderr)
	Eventually(session, 10*time.Minute).Should(gexec.Exit(0))
}

func (b BBL) fetchValue(value string) string {
	args := []string{
		"--state-dir", b.stateDirectory,
		value,
	}

	stdout := bytes.NewBuffer([]byte{})
	stderr := bytes.NewBuffer([]byte{})
	b.execute(args, stdout, stderr).Wait()

	return strings.TrimSpace(string(stdout.Bytes()))
}

func (b BBL) execute(args []string, stdout io.Writer, stderr io.Writer) *gexec.Session {
	cmd := exec.Command(b.pathToBBL, args...)
	session, err := gexec.Start(cmd, stdout, stderr)
	Expect(err).NotTo(HaveOccurred())

	return session
}
