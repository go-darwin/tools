// Copyright 2019 The go-darwin Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"fmt"
	"strings"
)

// Service is list of not important services.
//
// from:
//   https://gist.github.com/pwnsdx/1217727ca57de2dd2a372afdd7a0fc21
var Services = [...]string{
	// iCloud
	"com.apple.security.cloudkeychainproxy3",
	"com.apple.iCloudUserNotifications",
	"com.apple.icloud.findmydeviced.findmydevice-user-agent",
	"com.apple.icloud.fmfd",
	"com.apple.icloud.searchpartyuseragent",
	"com.apple.cloudd",
	"com.apple.cloudpaird",
	"com.apple.cloudphotod",
	"com.apple.followupd",
	"com.apple.protectedcloudstorage.protectedcloudkeysyncing",

	// Safari useless stuff
	"com.apple.SafariBookmarksSyncAgent",
	"com.apple.SafariCloudHistoryPushAgent",
	"com.apple.WebKit.PluginAgent",

	// iMessage / Facetime
	"com.apple.imagent",
	"com.apple.imautomatichistorydeletionagent",
	"com.apple.imklaunchagent",
	"com.apple.imtransferagent",
	"com.apple.avconferenced",

	// Game Center / Passbook / Apple TV / Homekit...
	"com.apple.gamed",
	"com.apple.passd",
	"com.apple.Maps.pushdaemon",
	"com.apple.videosubscriptionsd",
	"com.apple.CommCenter-osx",
	"com.apple.homed",

	// Ad-related
	"com.apple.ap.adprivacyd",
	"com.apple.ap.adservicesd",

	// Screensharing
	"com.apple.screensharing.MessagesAgent",
	"com.apple.screensharing.agent",
	"com.apple.screensharing.menuextra",

	// Siri
	"com.apple.siriknowledged",
	"com.apple.assistant_service",
	"com.apple.assistantd",
	"com.apple.Siri.agent",
	"com.apple.parsec-fbf",

	// VoiceOver / accessibility-related stuff
	"com.apple.VoiceOver",
	"com.apple.voicememod",
	"com.apple.accessibility.AXVisualSupportAgent",
	"com.apple.accessibility.dfrhud",
	"com.apple.accessibility.heard",

	// Quicklook
	"com.apple.quicklook.ui.helper",
	"com.apple.quicklook.ThumbnailsAgent",
	"com.apple.quicklook",

	// Sidecar
	"com.apple.sidecar-hid-relay",
	"com.apple.sidecar-relay",

	// Debugging process
	"com.apple.spindump_agent",
	"com.apple.ReportCrash",
	"com.apple.ReportGPURestart",
	"com.apple.ReportPanic",
	"com.apple.DiagnosticReportCleanup",
	"com.apple.TrustEvaluationAgent",

	// Screentime
	"com.apple.ScreenTimeAgent",
	"com.apple.UsageTrackingAgent",

	// Others
	"com.apple.telephonyutilities.callservicesd",
	"com.apple.photoanalysisd",
	"com.apple.parsecd",
	"com.apple.AOSPushRelay",
	"com.apple.AOSHeartbeat",
	"com.apple.AirPlayUIAgent",
	"com.apple.AirPortBaseStationAgent",
	"com.apple.familycircled",
	"com.apple.familycontrols.useragent",
	"com.apple.familynotificationd",
	"com.apple.findmymacmessenger",
	"com.apple.sharingd",
	"com.apple.identityservicesd",
	"com.apple.java.InstallOnDemand",
	"com.apple.parentalcontrols.check",
	"com.apple.security.keychain-circle-notification",
	"com.apple.syncdefaultsd",
	"com.apple.appleseed.seedusaged",
	"com.apple.appleseed.seedusaged.postinstall",
	"com.apple.CallHistorySyncHelper",
	"com.apple.RemoteDesktop",
	"com.apple.CallHistoryPluginHelper",
	"com.apple.SocialPushAgent",
	"com.apple.touristd",
	"com.apple.macos.studentd",
	"com.apple.KeyboardAccessAgent",
	"com.apple.exchange.exchangesyncd",
	"com.apple.suggestd",
	"com.apple.AddressBook.abd",
	"com.apple.helpd",
	"com.apple.amp.mediasharingd",
	"com.apple.mediaanalysisd",
	"com.apple.mediaremoteagent",
	"com.apple.remindd",

	// iCloud
	"com.apple.analyticsd",

	// Others
	"com.apple.netbiosd",
	"com.apple.preferences.timezone.admintool",
	"com.apple.remotepairtool",
	"com.apple.security.FDERecoveryAgent",
	"com.apple.SubmitDiagInfo",
	"com.apple.screensharing",
	"com.apple.appleseed.fbahelperd",
	"com.apple.apsd",
	"com.apple.ManagedClient.cloudconfigurationd",
	"com.apple.ManagedClient.enroll",
	"com.apple.ManagedClient",
	"com.apple.ManagedClient.startup",
	"com.apple.locate",
	"com.apple.locationd",
	"com.apple.eapolcfg_auth",
	"com.apple.RemoteDesktop.PrivilegeProxy",
	"com.apple.mediaremoted",

	// iCloud
	"com.apple.analyticsd",
}

func main() {
	fmt.Println(strings.Join(Services[:], "\n"))
}
