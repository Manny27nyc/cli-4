package v7_test

import (
	"code.cloudfoundry.org/cli/actor/actionerror"
	"code.cloudfoundry.org/cli/actor/v7action"
	"code.cloudfoundry.org/cli/command/commandfakes"
	. "code.cloudfoundry.org/cli/command/v7"
	"code.cloudfoundry.org/cli/command/v7/v7fakes"
	"code.cloudfoundry.org/cli/resources"
	"code.cloudfoundry.org/cli/types"
	"code.cloudfoundry.org/cli/util/configv3"
	"code.cloudfoundry.org/cli/util/ui"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gbytes"
)

var _ = Describe("Space Quota Command", func() {
	var (
		cmd             SpaceQuotaCommand
		testUI          *ui.UI
		fakeConfig      *commandfakes.FakeConfig
		fakeSharedActor *commandfakes.FakeSharedActor
		fakeActor       *v7fakes.FakeActor
		executeErr      error
	)

	BeforeEach(func() {
		testUI = ui.NewTestUI(nil, NewBuffer(), NewBuffer())
		fakeConfig = new(commandfakes.FakeConfig)
		fakeSharedActor = new(commandfakes.FakeSharedActor)
		fakeActor = new(v7fakes.FakeActor)
		fakeConfig.TargetedOrganizationNameReturns("some-org")

		cmd = SpaceQuotaCommand{
			BaseCommand: BaseCommand{
				UI:          testUI,
				Config:      fakeConfig,
				SharedActor: fakeSharedActor,
				Actor:       fakeActor,
			},
		}

		cmd.RequiredArgs.SpaceQuota = "some-space-quota"
	})

	JustBeforeEach(func() {
		executeErr = cmd.Execute(nil)
	})

	When("checking the target fails", func() {
		BeforeEach(func() {
			fakeSharedActor.CheckTargetReturns(
				actionerror.NotLoggedInError{BinaryName: "binaryName"})
		})

		It("returns an error", func() {
			Expect(executeErr).To(MatchError(actionerror.NotLoggedInError{BinaryName: "binaryName"}))

			Expect(fakeSharedActor.CheckTargetCallCount()).To(Equal(1))
			targetedOrganizationRequired, targetedSpaceRequired := fakeSharedActor.CheckTargetArgsForCall(0)
			Expect(targetedOrganizationRequired).To(Equal(true))
			Expect(targetedSpaceRequired).To(Equal(false))
		})
	})

	When("getting the space quota fails", func() {
		BeforeEach(func() {
			fakeActor.GetCurrentUserReturns(
				configv3.User{
					Name: "some-user",
				},
				nil)

			fakeActor.GetSpaceQuotaByNameReturns(
				resources.SpaceQuota{},
				v7action.Warnings{"warning-1", "warning-2"},
				actionerror.SpaceQuotaNotFoundError{})
		})

		It("returns a translatable error and outputs all warnings", func() {
			Expect(testUI.Out).To(Say("Getting space quota some-space-quota for org some-org as some-user..."))

			Expect(executeErr).To(MatchError(actionerror.SpaceQuotaNotFoundError{}))
			Expect(fakeActor.GetSpaceQuotaByNameCallCount()).To(Equal(1))
			Expect(testUI.Err).To(Say("warning-1"))
			Expect(testUI.Err).To(Say("warning-2"))
		})
	})

	When("getting the space quota succeeds", func() {
		BeforeEach(func() {
			fakeActor.GetCurrentUserReturns(configv3.User{Name: "some-user"}, nil)
			fakeConfig.TargetedOrganizationReturns(configv3.Organization{Name: "some-org", GUID: "some-org-guid"})

			falseValue := false
			fakeActor.GetSpaceQuotaByNameReturns(
				resources.SpaceQuota{
					Quota: resources.Quota{
						Name: "some-space-quota",
						Apps: resources.AppLimit{
							TotalMemory:       &types.NullInt{IsSet: true, Value: 2048},
							InstanceMemory:    &types.NullInt{IsSet: true, Value: 1024},
							TotalAppInstances: &types.NullInt{IsSet: true, Value: 2},
						},
						Services: resources.ServiceLimit{
							TotalServiceInstances: &types.NullInt{IsSet: false},
							PaidServicePlans:      &falseValue,
						},
						Routes: resources.RouteLimit{
							TotalRoutes:        &types.NullInt{IsSet: true, Value: 4},
							TotalReservedPorts: &types.NullInt{IsSet: false},
						},
					},
				},
				v7action.Warnings{"warning-1", "warning-2"},
				nil)
		})

		It("displays the quota and all warnings", func() {
			Expect(executeErr).ToNot(HaveOccurred())
			Expect(fakeActor.GetSpaceQuotaByNameCallCount()).To(Equal(1))
			quotaName, orgGUID := fakeActor.GetSpaceQuotaByNameArgsForCall(0)
			Expect(quotaName).To(Equal("some-space-quota"))
			Expect(orgGUID).To(Equal("some-org-guid"))

			Expect(testUI.Out).To(Say("Getting space quota some-space-quota for org some-org as some-user..."))
			Expect(testUI.Err).To(Say("warning-1"))
			Expect(testUI.Err).To(Say("warning-2"))

			Expect(testUI.Out).To(Say(`total memory:\s+2G`))
			Expect(testUI.Out).To(Say(`instance memory:\s+1G`))
			Expect(testUI.Out).To(Say(`routes:\s+4`))
			Expect(testUI.Out).To(Say(`service instances:\s+unlimited`))
			Expect(testUI.Out).To(Say(`paid service plans:\s+disallowed`))
			Expect(testUI.Out).To(Say(`app instances:\s+2`))
			Expect(testUI.Out).To(Say(`route ports:\s+unlimited`))
		})
	})
})
