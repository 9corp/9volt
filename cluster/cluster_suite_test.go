package cluster

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/Sirupsen/logrus"
	"testing"
)

func TestConfig(t *testing.T) {
	logrus.SetLevel(logrus.FatalLevel)
	RegisterFailHandler(Fail)
	RunSpecs(t, "Cluster Suite")
}
