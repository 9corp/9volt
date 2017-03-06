package monitor

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/9corp/9volt/util"
	log "github.com/Sirupsen/logrus"
	resolver "github.com/miekg/dns"
)

const (
	DEFAULT_DNS_TIMEOUT     = time.Duration(2) * time.Second
	DEFAULT_DNS_RECORD_TYPE = "A"
)

type DnsMonitor struct {
	Base
	Timeout    time.Duration
	Client     *resolver.Client
	Expect     *regexp.Regexp
	RecordType string
}

func NewDnsMonitor(rmc *RootMonitorConfig) *DnsMonitor {
	dns := &DnsMonitor{
		Base: Base{
			RMC:        rmc,
			Identifier: "dns",
		},
		Client:     &resolver.Client{},
		Timeout:    DEFAULT_DNS_TIMEOUT,
		RecordType: "A",
	}

	// Override the default with the setting if we have one
	if rmc.Config.Timeout != util.CustomDuration(0) {
		dns.Timeout = time.Duration(rmc.Config.Timeout)
	}

	// We'll just use the cumulative timeout
	// https://godoc.org/github.com/miekg/dns#Client
	dns.Client.Timeout = dns.Timeout

	if len(rmc.Config.DnsRecordType) > 0 {
		dns.RecordType = rmc.Config.DnsRecordType
	}

	if len(rmc.Config.Expect) > 0 {
		dns.Expect = regexp.MustCompile(rmc.Config.Expect)
	} else {
		dns.Expect = regexp.MustCompile(".") // Match any non-empty
	}

	dns.MonitorFunc = dns.dnsCheck

	return dns
}

func (dns *DnsMonitor) Validate() error {
	log.Debugf(
		"%v: Performing monitor config validation for %v",
		dns.Identifier, dns.RMC.ConfigName,
	)

	if _, ok := resolver.StringToType[strings.ToUpper(dns.RMC.Config.DnsRecordType)]; !ok {
		return fmt.Errorf("Unknown record type: %s", dns.RMC.Config.DnsRecordType)
	}

	if len(dns.RMC.Config.DnsTarget) < 1 {
		return fmt.Errorf("No DNS target configured!")
	}

	if dns.Timeout >= time.Duration(dns.RMC.Config.Interval) {
		return fmt.Errorf(
			"'timeout' (%v) cannot equal or exceed 'interval' (%v)",
			dns.Timeout.String(), dns.RMC.Config.Interval.String(),
		)
	}

	if len(dns.RMC.Config.Expect) > 0 {
		_, err := regexp.Compile(dns.RMC.Config.Expect)
		if err != nil {
			return fmt.Errorf("Unable to compile DNS check regexp! (%s)", err.Error())
		}
	}

	return nil
}

// Do a full DNS check on a particular hostname against a particular
// DNS server. In this check 'target' is the thing we're looking up
// and 'host' is the DNS server we're talking to do the looking.
func (dns *DnsMonitor) dnsCheck() error {
	msg := &resolver.Msg{}

	qType, ok := resolver.StringToType[strings.ToUpper(dns.RecordType)]
	if !ok {
		return fmt.Errorf("Unknown record type: %s! Aborting.", dns.RecordType)
	}

	target := resolver.Fqdn(dns.RMC.Config.DnsTarget)
	server := dns.RMC.Config.Host + ":53" // Only support port 53 at least for now

	log.Debugf("%s: Resolving %s against %s", dns.Identifier, target, server)

	msg.SetQuestion(target, qType)

	resp, elapsed, err := dns.Client.Exchange(msg, server)
	if err != nil {
		return fmt.Errorf("DNS query failed: %s", err.Error())
	}

	expectedCount := dns.RMC.Config.DnsExpectedCount

	// If we didn't set an expectation, we don't check
	if expectedCount != 0 && len(resp.Answer) != expectedCount {
		return fmt.Errorf(
			"Unexpected result count (%d) received for %s against DNS server %s",
			len(resp.Answer), dns.RMC.Config.DnsTarget, dns.RMC.Config.Host,
		)
	}

	// In any case, no results is bad news
	if len(resp.Answer) == 0 {
		return fmt.Errorf(
			"No results received for %s against DNS server %s",
			dns.RMC.Config.DnsTarget, dns.RMC.Config.Host,
		)
	}

	// If we took too long, we fail
	if dns.RMC.Config.DnsMaxTime > util.CustomDuration(0) &&
		elapsed > time.Duration(dns.RMC.Config.DnsMaxTime) {

		return fmt.Errorf(
			"DNS check of %s against %s server took longer than %s allowed: %s",
			dns.RMC.Config.DnsTarget, dns.RMC.Config.Host,
			time.Duration(dns.RMC.Config.DnsMaxTime), elapsed,
		)
	}

	// So... did we get the expected record(s)?
	foundCount := 0
	for _, ans := range resp.Answer {
		// Regex on the whole returned record string
		matchStr := fmt.Sprintf("%v %v\n", ans.Header().String(), resolver.Field(ans, 1))
		if dns.Expect.Match([]byte(matchStr)) {
			foundCount += 1
			if expectedCount <= 1 {
				// Success, bail early
				return nil
			}
			continue
		}

		log.Warnf("%s: Found non-matching records in DNS result: '%s'",
			dns.Identifier, ans.Header().String(),
		)
	}

	if foundCount == expectedCount {
		return nil
	}

	if foundCount > 0 && foundCount < expectedCount {
		return fmt.Errorf(
			"DNS check of %s against %s had %d records. Only %d matched",
			dns.RMC.Config.DnsTarget, dns.RMC.Config.Host,
			len(resp.Answer), foundCount,
		)
	}

	return fmt.Errorf(
		"DNS check of %s against %s had %d records. None matched expected regex '%s'",
		dns.RMC.Config.DnsTarget, dns.RMC.Config.Host,
		len(resp.Answer), dns.RMC.Config.Expect,
	)
}
