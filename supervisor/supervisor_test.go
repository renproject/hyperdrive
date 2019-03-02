package supervisor_test

import (
	"time"

	"github.com/renproject/hyperdrive/supervisor"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Supervisor", func() {

	Context("when stopping the supervisor", func() {

		Context("when using the `RestartNone` policy", func() {

			It("should terminate", func() {
				sv := supervisor.New(supervisor.RestartNone)

				go func() {
					time.Sleep(10 * time.Millisecond)
					sv.Stop()
				}()
				sv.Start()

				Eventually(sv.Restarts()).Should(Receive(Equal(supervisor.ErrSupervisorStopped)))
			})
		})

		Context("when using the `RestartAll` policy", func() {

			It("should terminate", func() {
				sv := supervisor.New(supervisor.RestartAll)

				go func() {
					time.Sleep(10 * time.Millisecond)
					sv.Stop()
				}()
				sv.Start()

				Eventually(sv.Restarts()).Should(Receive(Equal(supervisor.ErrSupervisorStopped)))
			})
		})
	})

	Context("when spawning runners into the supervisor", func() {

		Context("when using the `RestartNone` policy", func() {

			It("should stop after panicking", func() {
				sv := supervisor.New(supervisor.RestartNone)

				sv.SpawnFunc(func(ctx supervisor.Context) {
					time.Sleep(time.Millisecond)
					panic("this panic will be recovered by the supervisor")
				})
				sv.Start()

				var err error
				Eventually(sv.Restarts()).Should(Receive(&err))
				Expect(err.Error()).To(ContainSubstring("this panic will be recovered by the supervisor"))

				Eventually(sv.Restarts()).Should(Receive(Equal(supervisor.ErrSupervisorStopped)))
			})
		})

		Context("when using the `RestartAll` policy", func() {

			It("should continue after panicking", func() {
				sv := supervisor.New(supervisor.RestartAll)

				go func() {
					time.Sleep(10 * time.Millisecond)
					sv.Stop()
				}()
				sv.SpawnFunc(func(ctx supervisor.Context) {
					time.Sleep(time.Millisecond)
					panic("this panic will be recovered by the supervisor")
				})
				sv.Start()

				var err error
				Eventually(sv.Restarts()).Should(Receive(&err))
				Expect(err.Error()).To(ContainSubstring("this panic will be recovered by the supervisor"))

				Eventually(sv.Restarts()).Should(Receive(Equal(supervisor.ErrSupervisorStopped)))
			})
		})
	})
})
