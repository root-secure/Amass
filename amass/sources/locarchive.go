// Copyright 2017 Jeff Foley. All rights reserved.
// Use of this source code is governed by Apache 2 LICENSE that can be found in the LICENSE file.

package sources

import (
	"github.com/root-secure/Amass/amass/core"
	"github.com/root-secure/Amass/amass/utils"
)

// LoCArchive is the Service that handles access to the LoCArchive data source.
type LoCArchive struct {
	core.BaseService

	domain     string
	baseURL    string
	SourceType string
	filter     *utils.StringFilter
}

// NewLoCArchive returns he object initialized, but not yet started.
func NewLoCArchive(config *core.Config, bus *core.EventBus) *LoCArchive {
	l := &LoCArchive{
		domain:     "webarchive.loc.gov",
		baseURL:    "http://webarchive.loc.gov/all",
		SourceType: core.ARCHIVE,
		filter:     utils.NewStringFilter(),
	}

	l.BaseService = *core.NewBaseService(l, "LoCArchive", config, bus)
	return l
}

// OnStart implements the Service interface
func (l *LoCArchive) OnStart() error {
	l.BaseService.OnStart()

	l.Bus().Subscribe(core.NameResolvedTopic, l.SendDNSRequest)
	go l.processRequests()
	return nil
}

func (l *LoCArchive) processRequests() {
	for {
		select {
		case <-l.Quit():
			return
		case req := <-l.DNSRequestChan():
			if l.Config().IsDomainInScope(req.Name) {
				l.executeQuery(req.Name, req.Domain)
			}
		case <-l.AddrRequestChan():
		case <-l.ASNRequestChan():
		case <-l.WhoisRequestChan():
		}
	}
}

func (l *LoCArchive) executeQuery(sn, domain string) {
	if sn == "" || domain == "" {
		return
	}
	if l.filter.Duplicate(sn) {
		return
	}

	names, err := crawl(l, l.baseURL, l.domain, sn, domain)
	if err != nil {
		l.Config().Log.Printf("%s: %v", l.String(), err)
		return
	}

	for _, name := range names {
		l.Bus().Publish(core.NewNameTopic, &core.DNSRequest{
			Name:   cleanName(name),
			Domain: domain,
			Tag:    l.SourceType,
			Source: l.String(),
		})
	}
}
