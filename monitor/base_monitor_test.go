package monitor

import (
	"errors"
	"time"

	"github.com/9corp/9volt/alerter"
	"github.com/9corp/9volt/state"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

const (
	MaxMessages      int = 4
	CriticalMessages int = 2
	WarningMessages  int = 1
	ResolveMessages  int = 3
)

var _ = Describe("base_monitor", func() {
	var (
		monitor     *Base
		rootConfig  *RootMonitorConfig
		tickerChan  chan time.Time
		stopChan    chan bool
		stateChan   chan *state.Message
		messageChan chan *alerter.Message
	)
	BeforeEach(func() {
		tickerChan = make(chan time.Time, MaxMessages)
		stopChan = make(chan bool, 1)
		stateChan = make(chan *state.Message, MaxMessages)
		messageChan = make(chan *alerter.Message, MaxMessages)
		mockTicker := &time.Ticker{
			C: tickerChan,
		}

		rootConfig = &RootMonitorConfig{
			Ticker:         mockTicker,
			StopChannel:    stopChan,
			StateChannel:   stateChan,
			ConfigName:     "mock_config",
			MessageChannel: messageChan,
			Config: &MonitorConfig{
				CriticalThreshold: CriticalMessages,
				WarningThreshold:  WarningMessages,
			},
		}
		monitor = &Base{
			Identifier: "dummy_base",
			RMC:        rootConfig,
		}
	})
	Context("Identify", func() {
		It("returns valid identifier for monitor", func() {
			Expect(monitor.Identify()).To(Equal("dummy_base"))
		})
	})

	Context("Stop", func() {
		It("sends true to a stop channel", func() {
			monitor.Stop()
			Eventually(stopChan).Should(Receive())
		})
	})

	Context("Run", func() {
		It("stops the main loop when stop is called", func() {
			monitor.Stop()
			Expect(monitor.Run()).To(BeNil())
		})
		Context("successful check", func() {
			var successfulCheck func() error
			BeforeEach(func() {
				successfulCheck = func() error {
					monitor.Stop()
					return nil
				}
				tickerChan <- time.Now()
				monitor.MonitorFunc = successfulCheck
				monitor.Run()
			})

			It("logs ok state to a RMC.StateChannel", func() {
				var receivedState *state.Message
				Eventually(monitor.RMC.StateChannel).Should(Receive(&receivedState))
				Expect(receivedState.Status).To(Equal("ok"))
				Expect(receivedState.Check).To(Equal("mock_config"))
				Expect(receivedState.Count).To(Equal(0))
				Expect(receivedState.Message).To(Equal("N/A"))
			})
		})

		Context("warning", func() {
			var failedCheck func() error
			BeforeEach(func() {
				var loops int = 0
				failedCheck = func() error {
					loops++
					if loops >= WarningMessages {
						monitor.Stop()
					}
					return errors.New("Failed check")
				}
				for i := 0; i < WarningMessages; i++ {
					tickerChan <- time.Now()
				}
				monitor.MonitorFunc = failedCheck
				monitor.Run()
			})

			It("logs warning state to RMC.StateChannel", func() {
				var receivedState *state.Message
				Eventually(monitor.RMC.StateChannel).Should(Receive(&receivedState))
				Expect(receivedState.Status).To(Equal("warning"))
				Expect(receivedState.Count).To(Equal(1))
			})

			It("sends warning to alerter", func() {
				var receivedAlert *alerter.Message
				Eventually(monitor.RMC.MessageChannel).Should(Receive(&receivedAlert))
				Expect(receivedAlert.Type).To(Equal("warning"))
				Expect(receivedAlert.Title).To(ContainSubstring("DUMMY_BASE check 'mock_config' failure"))
				Expect(receivedAlert.Text).To(ContainSubstring("entered into warning state after 1 checks"))
				Expect(receivedAlert.Source).To(Equal("mock_config"))
				Expect(receivedAlert.Contents["WarningThreshold"]).To(Equal("1"))
				Expect(receivedAlert.Contents["ErrorDetails"]).To(Equal("Failed check"))
			})
		})

		Context("critical", func() {
			BeforeEach(func() {
				var failedCheck func() error
				var loops int = 0
				failedCheck = func() error {
					loops++
					if loops >= CriticalMessages {
						monitor.Stop()
					}
					return errors.New("Failed check")
				}
				for i := 0; i < CriticalMessages; i++ {
					tickerChan <- time.Now()
				}
				monitor.MonitorFunc = failedCheck
				monitor.Run()
			})

			It("logs critical state to RMC.StateChannel", func() {
				var receivedState *state.Message

				for i := 0; i < CriticalMessages; i++ {
					Eventually(monitor.RMC.StateChannel).Should(Receive(&receivedState))
				}
				Expect(receivedState.Status).To(Equal("critical"))
				Expect(receivedState.Count).To(Equal(2))
			})

			It("sends critical to alerter", func() {
				var receivedAlert *alerter.Message
				for i := 0; i < CriticalMessages; i++ {
					Eventually(monitor.RMC.MessageChannel).Should(Receive(&receivedAlert))
				}
				Expect(receivedAlert.Type).To(Equal("critical"))
				Expect(receivedAlert.Title).To(ContainSubstring("DUMMY_BASE check 'mock_config' failure"))
				Expect(receivedAlert.Text).To(ContainSubstring("entered into critical state after 2 checks"))
				Expect(receivedAlert.Source).To(Equal("mock_config"))
				Expect(receivedAlert.Contents["WarningThreshold"]).To(Equal("1"))
				Expect(receivedAlert.Contents["ErrorDetails"]).To(Equal("Failed check"))
			})
		})

		Context("resolve after warning", func() {
			var warningResolve func() error
			BeforeEach(func() {
				loops := 0
				warningResolve = func() error {
					loops++
					if loops >= WarningMessages+1 {
						monitor.Stop()
					}
					if loops <= WarningMessages {
						return errors.New("failed check")
					}
					return nil
				}

				for i := 0; i < WarningMessages+1; i++ {
					tickerChan <- time.Now()
				}

				monitor.MonitorFunc = warningResolve

				monitor.Run()
			})

			It("resolves alert after a warning state is issued", func() {
				var receivedAlert *alerter.Message
				for i := 0; i < WarningMessages+1; i++ {
					Eventually(monitor.RMC.MessageChannel).Should(Receive(&receivedAlert))
				}
				Expect(receivedAlert.Type).To(Equal("resolve"))
			})

			It("logs state as ok", func() {
				var receivedState *state.Message
				for i := 0; i < WarningMessages+1; i++ {
					Eventually(monitor.RMC.StateChannel).Should(Receive(&receivedState))
				}
				Expect(receivedState.Status).To(Equal("ok"))

			})

		})

		Context("resolve after critical", func() {
			var criticalResolve func() error
			BeforeEach(func() {
				loops := 0
				criticalResolve = func() error {
					loops++
					if loops >= CriticalMessages+1 {
						monitor.Stop()
					}
					if loops <= CriticalMessages {
						return errors.New("failed check")
					}
					return nil
				}

				for i := 0; i < CriticalMessages+1; i++ {
					tickerChan <- time.Now()
				}

				monitor.MonitorFunc = criticalResolve

				monitor.Run()
			})

			It("resolves alert after a critical state is issued", func() {
				var receivedAlert *alerter.Message
				for i := 0; i < CriticalMessages+1; i++ {
					Eventually(monitor.RMC.MessageChannel).Should(Receive(&receivedAlert))
				}
				Expect(receivedAlert.Type).To(Equal("resolve"))
			})

			It("logs state as ok", func() {
				var receivedState *state.Message
				for i := 0; i < CriticalMessages+1; i++ {
					Eventually(monitor.RMC.StateChannel).Should(Receive(&receivedState))
				}
				Expect(receivedState.Status).To(Equal("ok"))

			})
		})
	})
})
