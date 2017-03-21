package dal

import (
	"testing"

	"github.com/Sirupsen/logrus"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestDALSuite(t *testing.T) {
	// reduce the noise when testing
	logrus.SetLevel(logrus.FatalLevel)

	RegisterFailHandler(Fail)
	RunSpecs(t, "DAL Suite")
}
