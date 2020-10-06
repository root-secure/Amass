// Copyright 2017 Jeff Foley. All rights reserved.
// Use of this source code is governed by Apache 2 LICENSE that can be found in the LICENSE file.

package sources

import (
	"github.com/root-secure/Amass/amass/core"
	"github.com/root-secure/Amass/amass/utils"
)

// ArchiveIt is the Service that handles access to the ArchiveIt data source.
type ArchiveIt struct {
	core.BaseService

	domain     string
	baseURL    string
	SourceType string
	filter     *utils.StringFilter
}

// NewArchiveIt returns he object initialized, but not yet started.
func NewArchiveIt(config *core.Config, bus *core.EventBus) *ArchiveIt {
	a := &ArchiveIt{
		domain:     "wayback.archive-it.org",
		baseURL:    "https://wayback.archive-it.org/all",
		SourceType: core.ARCHIVE,
		filter:     utils.NewStringFilter(),
	}

	a.BaseService = *core.NewBaseService(a, "ArchiveIt", config, bus)
	return a
}

// OnStart implements the Service interface
func (a *ArchiveIt) OnStart() error {
	a.BaseService.OnStart()

	a.Bus().Subscribe(core.NameResolvedTopic, a.SendDNSRequest)
	go a.processRequests()
	return nil
}

func (a *ArchiveIt) processRequests() {
	for {
		select {
		case <-a.Quit():
			return
		case req := <-a.DNSRequestChan():
			if a.Config().IsDomainInScope(req.Name) {
				a.executeQuery(req.Name, req.Domain)
			}
		case <-a.AddrRequestChan():
		case <-a.ASNRequestChan():
		case <-a.WhoisRequestChan():
		}
	}
}

func (a *ArchiveIt) executeQuery(sn, domain string) {
	if sn == "" || domain == "" || a.filter.Duplicate(sn) {
		return
	}

	names, err := crawl(a, a.baseURL, a.domain, sn, domain)
	if err != nil {
		a.Config().Log.Printf("%s: %v", a.String(), err)
		return
	}

	for _, name := range names {
		a.Bus().Publish(core.NewNameTopic, &core.DNSRequest{
			Name:   cleanName(name),
			Domain: domain,
			Tag:    a.SourceType,
			Source: a.String(),
		})
	}
}
