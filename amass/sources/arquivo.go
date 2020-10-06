// Copyright 2017 Jeff Foley. All rights reserved.
// Use of this source code is governed by Apache 2 LICENSE that can be found in the LICENSE file.

package sources

import (
	"github.com/root-secure/Amass/amass/core"
	"github.com/root-secure/Amass/amass/utils"
)

// Arquivo is the Service that handles access to the Arquivo data source.
type Arquivo struct {
	core.BaseService

	domain     string
	baseURL    string
	SourceType string
	filter     *utils.StringFilter
}

// NewArquivo returns he object initialized, but not yet started.
func NewArquivo(config *core.Config, bus *core.EventBus) *Arquivo {
	a := &Arquivo{
		domain:     "arquivo.pt",
		baseURL:    "http://arquivo.pt/wayback",
		SourceType: core.ARCHIVE,
		filter:     utils.NewStringFilter(),
	}

	a.BaseService = *core.NewBaseService(a, "Arquivo", config, bus)
	return a
}

// OnStart implements the Service interface
func (a *Arquivo) OnStart() error {
	a.BaseService.OnStart()

	a.Bus().Subscribe(core.NameResolvedTopic, a.SendDNSRequest)
	go a.processRequests()
	return nil
}

func (a *Arquivo) processRequests() {
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

func (a *Arquivo) executeQuery(sn, domain string) {
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
