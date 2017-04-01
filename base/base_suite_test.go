package base

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/Sirupsen/logrus"
	"testing"
)

func TestBase(t *testing.T) {
	logrus.SetLevel(logrus.FatalLevel)
	RegisterFailHandler(Fail)
	RunSpecs(t, "Base Suite")
}
