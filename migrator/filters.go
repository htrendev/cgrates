/*
Real-time Online/Offline Charging System (OCS) for Telecom & ISP environments
Copyright (C) ITsysCOM GmbH

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU General Public License for more details.

You should have received a copy of the GNU General Public License
along with this program.  If not, see <http://www.gnu.org/licenses/>
*/

package migrator

import (
	"fmt"
	"strings"

	"github.com/cgrates/cgrates/config"
	"github.com/cgrates/cgrates/engine"
	"github.com/cgrates/cgrates/utils"
)

func (m *Migrator) migrateCurrentRequestFilter() (err error) {
	var ids []string
	ids, err = m.dmIN.DataManager().DataDB().GetKeysForPrefix(utils.FilterPrefix)
	if err != nil {
		return err
	}
	for _, id := range ids {
		tntID := strings.SplitN(strings.TrimPrefix(id, utils.FilterPrefix), utils.InInFieldSep, 2)
		if len(tntID) < 2 {
			return fmt.Errorf("Invalid key <%s> when migrating filters", id)
		}
		fl, err := m.dmIN.DataManager().GetFilter(tntID[0], tntID[1], false, false, utils.NonTransactional)
		if err != nil {
			return err
		}
		if m.dryRun || fl == nil {
			continue
		}
		if err := m.dmOut.DataManager().SetFilter(fl, true); err != nil {
			return err
		}
		if err := m.dmIN.DataManager().RemoveFilter(tntID[0], tntID[1],
			utils.NonTransactional, true); err != nil {
			return err
		}
		m.stats[utils.RQF]++
	}
	return
}

var filterTypes = utils.NewStringSet([]string{utils.MetaRSR, utils.MetaStatS, utils.MetaResources,
	utils.MetaNotRSR, utils.MetaNotStatS, utils.MetaNotResources})

func migrateFilterV1(fl *v1Filter) (fltr *engine.Filter) {
	fltr = &engine.Filter{
		Tenant:             fl.Tenant,
		ID:                 fl.ID,
		Rules:              make([]*engine.FilterRule, len(fl.Rules)),
		ActivationInterval: fl.ActivationInterval,
	}
	for i, rule := range fl.Rules {
		fltr.Rules[i] = &engine.FilterRule{
			Type:    rule.Type,
			Element: rule.FieldName,
			Values:  rule.Values,
		}
		if rule.FieldName == "" ||
			strings.HasPrefix(rule.FieldName, utils.DynamicDataPrefix) ||
			filterTypes.Has(rule.Type) {
			continue
		}
		fltr.Rules[i].Element = utils.DynamicDataPrefix + utils.MetaReq + utils.NestingSep + rule.FieldName
	}
	return
}

func migrateFilterV2(fl *v1Filter) (fltr *engine.Filter) {
	fltr = &engine.Filter{
		Tenant:             fl.Tenant,
		ID:                 fl.ID,
		Rules:              make([]*engine.FilterRule, len(fl.Rules)),
		ActivationInterval: fl.ActivationInterval,
	}
	for i, rule := range fl.Rules {
		fltr.Rules[i] = &engine.FilterRule{
			Type:    rule.Type,
			Element: rule.FieldName,
			Values:  rule.Values,
		}
		if (rule.FieldName == "" && rule.Type != utils.MetaRSR) ||
			strings.HasPrefix(rule.FieldName, utils.DynamicDataPrefix+utils.MetaReq) ||
			strings.HasPrefix(rule.FieldName, utils.DynamicDataPrefix+utils.MetaVars) ||
			strings.HasPrefix(rule.FieldName, utils.DynamicDataPrefix+utils.MetaCgreq) ||
			strings.HasPrefix(rule.FieldName, utils.DynamicDataPrefix+utils.MetaCgrep) ||
			strings.HasPrefix(rule.FieldName, utils.DynamicDataPrefix+utils.MetaRep) ||
			strings.HasPrefix(rule.FieldName, utils.DynamicDataPrefix+utils.MetaCGRAReq) ||
			strings.HasPrefix(rule.FieldName, utils.DynamicDataPrefix+utils.MetaAct) {
			continue
		}
		if rule.Type != utils.MetaRSR {
			// in case we found dynamic data prefix we remove it
			if strings.HasPrefix(rule.FieldName, utils.DynamicDataPrefix) {
				fl.Rules[i].FieldName = fl.Rules[i].FieldName[1:]
			}
			fltr.Rules[i].Element = utils.DynamicDataPrefix + utils.MetaReq + utils.NestingSep + rule.FieldName
		} else {
			for idx, val := range rule.Values {
				if strings.HasPrefix(val, utils.DynamicDataPrefix) {
					// remove dynamic data prefix from fieldName
					val = val[1:]
				}
				fltr.Rules[i].Values[idx] = utils.DynamicDataPrefix + utils.MetaReq + utils.NestingSep + val
			}
		}
	}
	return
}

func migrateFilterV3(fl *v1Filter) (fltr *engine.Filter) {
	fltr = &engine.Filter{
		Tenant:             fl.Tenant,
		ID:                 fl.ID,
		Rules:              make([]*engine.FilterRule, len(fl.Rules)),
		ActivationInterval: fl.ActivationInterval,
	}
	for i, rule := range fl.Rules {
		fltr.Rules[i] = &engine.FilterRule{
			Type:    rule.Type,
			Element: rule.FieldName,
			Values:  rule.Values,
		}
	}
	return
}

func migrateInlineFilter(fl string) string {
	if fl == "" || !strings.HasPrefix(fl, utils.Meta) {
		return fl
	}
	ruleSplt := strings.Split(fl, utils.InInFieldSep)
	if len(ruleSplt) < 3 {
		return fl
	}

	if strings.HasPrefix(ruleSplt[1], utils.DynamicDataPrefix) ||
		filterTypes.Has(ruleSplt[0]) {
		return fl
	}
	return fmt.Sprintf("%s:~%s:%s", ruleSplt[0], utils.MetaReq+utils.NestingSep+ruleSplt[1], strings.Join(ruleSplt[2:], utils.InInFieldSep))
}

func migrateInlineFilterV2(fl string) string {
	if fl == "" || !strings.HasPrefix(fl, utils.Meta) {
		return fl
	}
	ruleSplt := strings.Split(fl, utils.InInFieldSep)
	if len(ruleSplt) < 3 {
		return fl
	}
	if ruleSplt[1] != utils.EmptyString && // no need conversion
		(strings.HasPrefix(ruleSplt[1], utils.DynamicDataPrefix+utils.MetaReq) ||
			strings.HasPrefix(ruleSplt[1], utils.DynamicDataPrefix+utils.MetaVars) ||
			strings.HasPrefix(ruleSplt[1], utils.DynamicDataPrefix+utils.MetaCgreq) ||
			strings.HasPrefix(ruleSplt[1], utils.DynamicDataPrefix+utils.MetaCgrep) ||
			strings.HasPrefix(ruleSplt[1], utils.DynamicDataPrefix+utils.MetaRep) ||
			strings.HasPrefix(ruleSplt[1], utils.DynamicDataPrefix+utils.MetaCGRAReq) ||
			strings.HasPrefix(ruleSplt[1], utils.DynamicDataPrefix+utils.MetaAct)) {
		return fl
	}

	if ruleSplt[0] != utils.MetaRSR {
		if strings.HasPrefix(ruleSplt[1], utils.DynamicDataPrefix) {
			// remove dynamic data prefix from fieldName
			ruleSplt[1] = ruleSplt[1][1:]
		}
		return fmt.Sprintf("%s:~%s:%s", ruleSplt[0], utils.MetaReq+utils.NestingSep+ruleSplt[1], strings.Join(ruleSplt[2:], utils.InInFieldSep))
	} // in case of *rsr filter we need to add the prefix at fieldValue
	if strings.HasPrefix(ruleSplt[2], utils.DynamicDataPrefix) {
		// remove dynamic data prefix from fieldName
		ruleSplt[2] = ruleSplt[2][1:]
	}
	return fmt.Sprintf("%s::~%s", ruleSplt[0], utils.MetaReq+utils.NestingSep+strings.Join(ruleSplt[2:], utils.InInFieldSep))
}

func (m *Migrator) migrateOthersv1() (err error) {
	if err = m.migrateResourceProfileFiltersV1(); err != nil {
		return err
	}
	if err = m.migrateStatQueueProfileFiltersV1(); err != nil {
		return err
	}
	if err = m.migrateThresholdsProfileFiltersV1(); err != nil {
		return err
	}
	if err = m.migrateSupplierProfileFiltersV1(); err != nil {
		return err
	}
	if err = m.migrateAttributeProfileFiltersV1(); err != nil {
		return err
	}
	if err = m.migrateChargerProfileFiltersV1(); err != nil {
		return err
	}
	if err = m.migrateDispatcherProfileFiltersV1(); err != nil {
		return err
	}
	return
}

func (m *Migrator) migrateRequestFilterV1() (fltr *engine.Filter, err error) {
	var v1Fltr *v1Filter
	if v1Fltr, err = m.dmIN.getV1Filter(); err != nil {
		return
	}
	if v1Fltr == nil {
		return
	}
	fltr = migrateFilterV1(v1Fltr)
	return
}

func (m *Migrator) migrateOthersV2() (err error) {
	if err = m.migrateResourceProfileFiltersV2(); err != nil {
		return fmt.Errorf("Error: <%s> when trying to migrate filter for ResourceProfiles",
			err.Error())
	}
	if err = m.migrateStatQueueProfileFiltersV2(); err != nil {
		return fmt.Errorf("Error: <%s> when trying to migrate filter for StatQueueProfiles",
			err.Error())
	}
	if err = m.migrateThresholdsProfileFiltersV2(); err != nil {
		return fmt.Errorf("Error: <%s> when trying to migrate filter for ThresholdProfiles",
			err.Error())
	}
	if err = m.migrateSupplierProfileFiltersV2(); err != nil {
		return fmt.Errorf("Error: <%s> when trying to migrate filter for SupplierProfiles",
			err.Error())
	}
	if err = m.migrateAttributeProfileFiltersV2(); err != nil {
		return fmt.Errorf("Error: <%s> when trying to migrate filter for AttributeProfiles",
			err.Error())
	}
	if err = m.migrateChargerProfileFiltersV2(); err != nil {
		return fmt.Errorf("Error: <%s> when trying to migrate filter for ChargerProfiles",
			err.Error())
	}
	if err = m.migrateDispatcherProfileFiltersV2(); err != nil {
		return fmt.Errorf("Error: <%s> when trying to migrate filter for DispatcherProfiles",
			err.Error())
	}
	return
}

func (m *Migrator) migrateRequestFilterV2() (fltr *engine.Filter, err error) {
	var v1Fltr *v1Filter
	if v1Fltr, err = m.dmIN.getV1Filter(); err != nil {
		return nil, err
	}
	if err == utils.ErrNoMoreData {
		return nil, nil
	}
	fltr = migrateFilterV2(v1Fltr)
	return
}

func (m *Migrator) migrateRequestFilterV3() (fltr *engine.Filter, err error) {
	var v1Fltr *v1Filter
	if v1Fltr, err = m.dmIN.getV1Filter(); err != nil {
		return nil, err
	}
	if v1Fltr == nil {
		return
	}
	fltr = migrateFilterV3(v1Fltr)
	return
}

func (m *Migrator) migrateFilters() (err error) {
	var vrs engine.Versions
	current := engine.CurrentDataDBVersions()
	if vrs, err = m.getVersions(utils.RQF); err != nil {
		return
	}
	migrated := true
	migratedFrom := 0
	var fltr *engine.Filter
	for {
		version := vrs[utils.RQF]
		for {
			switch version {
			case current[utils.RQF]:
				migrated = false
				if m.sameDataDB {
					break
				}
				if err = m.migrateCurrentRequestFilter(); err != nil {
					return err
				}
				version = 4
			case 1:
				if fltr, err = m.migrateRequestFilterV1(); err != nil && err != utils.ErrNoMoreData {
					return err
				}
				migratedFrom = 1
				version = 4
			case 2:
				if fltr, err = m.migrateRequestFilterV2(); err != nil && err != utils.ErrNoMoreData {
					return err
				}
				migratedFrom = 2
				version = 4
			case 3:
				if fltr, err = m.migrateRequestFilterV3(); err != nil && err != utils.ErrNoMoreData {
					return err
				}
				migratedFrom = 3
				version = 4
			}
			if version == current[utils.RQF] || err == utils.ErrNoMoreData {
				break
			}
		}
		if err == utils.ErrNoMoreData || !migrated {
			break
		}
		if !m.dryRun && migrated {
			//set filters
			switch migratedFrom {
			case 1:
				if err := m.dmOut.DataManager().SetFilter(fltr, true); err != nil {
					return fmt.Errorf("Error: <%s> when setting filter with tenant: <%s> and id: <%s> after migration",
						err.Error(), fltr.Tenant, fltr.ID)
				}
			case 2:
				if err := m.dmOut.DataManager().SetFilter(fltr, true); err != nil {
					return fmt.Errorf("Error: <%s> when setting filter with tenant: <%s> and id: <%s> after migration",
						err.Error(), fltr.Tenant, fltr.ID)
				}
			case 3:
				if err := m.dmOut.DataManager().SetFilter(fltr, true); err != nil {
					return fmt.Errorf("Error: <%s> when setting filter with tenant: <%s> and id: <%s> after migration",
						err.Error(), fltr.Tenant, fltr.ID)
				}
			}
		}
		m.stats[utils.RQF]++
	}
	if m.dryRun || !migrated {
		return nil
	}

	switch migratedFrom {
	case 1:
		if err := m.migrateOthersv1(); err != nil {
			return err
		}
	case 2:
		if err := m.migrateOthersV2(); err != nil {
			return err
		}
	}

	if err = m.setVersions(utils.RQF); err != nil {
		return err
	}
	return m.ensureIndexesDataDB(engine.ColFlt)
}

func (m *Migrator) migrateResourceProfileFiltersV1() (err error) {
	var ids []string
	ids, err = m.dmIN.DataManager().DataDB().GetKeysForPrefix(utils.ResourceProfilesPrefix)
	if err != nil {
		return err
	}
	for _, id := range ids {
		tntID := strings.SplitN(strings.TrimPrefix(id, utils.ResourceProfilesPrefix), utils.InInFieldSep, 2)
		if len(tntID) < 2 {
			return fmt.Errorf("Invalid key <%s> when migrating filter for resourceProfile", id)
		}
		res, err := m.dmIN.DataManager().GetResourceProfile(tntID[0], tntID[1], false, false, utils.NonTransactional)
		if err != nil {
			return err
		}
		if m.dryRun || res == nil {
			continue
		}
		for i, fl := range res.FilterIDs {
			res.FilterIDs[i] = migrateInlineFilter(fl)
		}
		if err := m.dmOut.DataManager().SetResourceProfile(res, true); err != nil {
			return err
		}
		m.stats[utils.RQF]++
	}
	return
}

func (m *Migrator) migrateStatQueueProfileFiltersV1() (err error) {
	var ids []string
	ids, err = m.dmIN.DataManager().DataDB().GetKeysForPrefix(utils.StatQueueProfilePrefix)
	if err != nil {
		return err
	}
	for _, id := range ids {
		tntID := strings.SplitN(strings.TrimPrefix(id, utils.StatQueueProfilePrefix), utils.InInFieldSep, 2)
		if len(tntID) < 2 {
			return fmt.Errorf("Invalid key <%s> when migrating filter for statQueueProfile", id)
		}
		sgs, err := m.dmIN.DataManager().GetStatQueueProfile(tntID[0], tntID[1], false, false, utils.NonTransactional)
		if err != nil {
			return err
		}
		if sgs == nil || m.dryRun {
			continue
		}
		for i, fl := range sgs.FilterIDs {
			sgs.FilterIDs[i] = migrateInlineFilter(fl)
		}
		if err = m.dmOut.DataManager().SetStatQueueProfile(sgs, true); err != nil {
			return err
		}
		m.stats[utils.RQF]++
	}
	return
}

func (m *Migrator) migrateThresholdsProfileFiltersV1() (err error) {
	var ids []string
	ids, err = m.dmIN.DataManager().DataDB().GetKeysForPrefix(utils.ThresholdProfilePrefix)
	if err != nil {
		return err
	}
	for _, id := range ids {
		tntID := strings.SplitN(strings.TrimPrefix(id, utils.ThresholdProfilePrefix), utils.InInFieldSep, 2)
		if len(tntID) < 2 {
			return fmt.Errorf("Invalid key <%s> when migrating filter for thresholdProfile", id)
		}
		ths, err := m.dmIN.DataManager().GetThresholdProfile(tntID[0], tntID[1], false, false, utils.NonTransactional)
		if err != nil {
			return err
		}
		if ths == nil || m.dryRun {
			continue
		}
		for i, fl := range ths.FilterIDs {
			ths.FilterIDs[i] = migrateInlineFilter(fl)
		}
		if err := m.dmOut.DataManager().SetThresholdProfile(ths, true); err != nil {
			return err
		}
		m.stats[utils.RQF]++
	}
	return
}

func (m *Migrator) migrateSupplierProfileFiltersV1() (err error) {
	for {
		var spp *SupplierProfile
		spp, err = m.dmIN.getSupplier()
		if err == utils.ErrNoMoreData {
			err = nil
			break
		}
		if err != nil {
			return err
		}
		if spp == nil || m.dryRun {
			continue
		}
		for i, fl := range spp.FilterIDs {
			spp.FilterIDs[i] = migrateInlineFilter(fl)
		}
		if err := m.dmOut.setSupplier(spp); err != nil {
			return err
		}
		m.stats[utils.RQF]++
	}
	return
}

func (m *Migrator) migrateAttributeProfileFiltersV1() (err error) {
	var ids []string
	ids, err = m.dmIN.DataManager().DataDB().GetKeysForPrefix(utils.AttributeProfilePrefix)
	if err != nil {
		return err
	}
	for _, id := range ids {
		tntID := strings.SplitN(strings.TrimPrefix(id, utils.AttributeProfilePrefix), utils.InInFieldSep, 2)
		if len(tntID) < 2 {
			return fmt.Errorf("Invalid key <%s> when migrating filter for attributeProfile", id)
		}
		attrPrf, err := m.dmIN.DataManager().GetAttributeProfile(tntID[0], tntID[1], false, false, utils.NonTransactional)
		if err != nil {
			return err
		}
		if attrPrf == nil || m.dryRun {
			continue
		}
		for i, fl := range attrPrf.FilterIDs {
			attrPrf.FilterIDs[i] = migrateInlineFilter(fl)
		}
		for i, attr := range attrPrf.Attributes {
			for j, fl := range attr.FilterIDs {
				attrPrf.Attributes[i].FilterIDs[j] = migrateInlineFilter(fl)
			}
		}
		if err := m.dmOut.DataManager().SetAttributeProfile(attrPrf, true); err != nil {
			return err
		}
		m.stats[utils.RQF]++
	}
	return
}

func (m *Migrator) migrateChargerProfileFiltersV1() (err error) {
	var ids []string
	ids, err = m.dmIN.DataManager().DataDB().GetKeysForPrefix(utils.ChargerProfilePrefix)
	if err != nil {
		return err
	}
	for _, id := range ids {
		tntID := strings.SplitN(strings.TrimPrefix(id, utils.ChargerProfilePrefix), utils.InInFieldSep, 2)
		if len(tntID) < 2 {
			return fmt.Errorf("Invalid key <%s> when migrating filter for chragerProfile", id)
		}
		cpp, err := m.dmIN.DataManager().GetChargerProfile(tntID[0], tntID[1], false, false, utils.NonTransactional)
		if err != nil {
			return err
		}
		if cpp == nil || m.dryRun {
			continue
		}
		for i, fl := range cpp.FilterIDs {
			cpp.FilterIDs[i] = migrateInlineFilter(fl)
		}
		if err := m.dmOut.DataManager().SetChargerProfile(cpp, true); err != nil {
			return err
		}
		m.stats[utils.RQF]++
	}
	return
}

func (m *Migrator) migrateDispatcherProfileFiltersV1() (err error) {
	var ids []string
	ids, err = m.dmIN.DataManager().DataDB().GetKeysForPrefix(utils.DispatcherProfilePrefix)
	if err != nil {
		return err
	}
	for _, id := range ids {
		tntID := strings.SplitN(strings.TrimPrefix(id, utils.DispatcherProfilePrefix), utils.InInFieldSep, 2)
		if len(tntID) < 2 {
			return fmt.Errorf("Invalid key <%s> when migrating filter for dispatcherProfile", id)
		}
		dpp, err := m.dmIN.DataManager().GetDispatcherProfile(tntID[0], tntID[1], false, false, utils.NonTransactional)
		if err != nil {
			return err
		}
		if dpp == nil || m.dryRun {
			continue
		}
		for i, fl := range dpp.FilterIDs {
			dpp.FilterIDs[i] = migrateInlineFilter(fl)
		}
		if err := m.dmOut.DataManager().SetDispatcherProfile(dpp, true); err != nil {
			return err
		}
		m.stats[utils.RQF]++
	}
	return
}

// migrate filters from v2 to v3 for items
func (m *Migrator) migrateResourceProfileFiltersV2() (err error) {
	var ids []string
	ids, err = m.dmIN.DataManager().DataDB().GetKeysForPrefix(utils.ResourceProfilesPrefix)
	if err != nil {
		return fmt.Errorf("error: <%s> when getting resource profile IDs", err.Error())
	}
	for _, id := range ids {
		tntID := strings.SplitN(strings.TrimPrefix(id, utils.ResourceProfilesPrefix), utils.InInFieldSep, 2)
		if len(tntID) < 2 {
			return fmt.Errorf("Invalid key <%s> when migrating filter for resourcerProfile", id)
		}
		res, err := m.dmIN.DataManager().GetResourceProfile(tntID[0], tntID[1], false, false, utils.NonTransactional)
		if err != nil {
			return fmt.Errorf("error: <%s> when getting resource profile with tenant: <%s> and id: <%s>",
				err.Error(), tntID[0], tntID[1])
		}
		if m.dryRun || res == nil {
			continue
		}
		for i, fl := range res.FilterIDs {
			res.FilterIDs[i] = migrateInlineFilterV2(fl)
		}
		if err := m.dmOut.DataManager().SetResourceProfile(res, true); err != nil {
			return fmt.Errorf("error: <%s> when setting resource profile with tenant: <%s> and id: <%s>",
				err.Error(), tntID[0], tntID[1])
		}
		m.stats[utils.RQF]++
	}
	return
}

func (m *Migrator) migrateStatQueueProfileFiltersV2() (err error) {
	var ids []string
	ids, err = m.dmIN.DataManager().DataDB().GetKeysForPrefix(utils.StatQueueProfilePrefix)
	if err != nil {
		return fmt.Errorf("error: <%s> when getting statQueue profile IDs", err.Error())
	}
	for _, id := range ids {
		tntID := strings.SplitN(strings.TrimPrefix(id, utils.StatQueueProfilePrefix), utils.InInFieldSep, 2)
		if len(tntID) < 2 {
			return fmt.Errorf("Invalid key <%s> when migrating filter for statQueueProfile", id)
		}
		sgs, err := m.dmIN.DataManager().GetStatQueueProfile(tntID[0], tntID[1], false, false, utils.NonTransactional)
		if err != nil {
			return fmt.Errorf("error: <%s> when getting statQueue profile with tenant: <%s> and id: <%s>",
				err.Error(), tntID[0], tntID[1])
		}
		if sgs == nil || m.dryRun {
			continue
		}
		for i, fl := range sgs.FilterIDs {
			sgs.FilterIDs[i] = migrateInlineFilterV2(fl)
		}
		if err = m.dmOut.DataManager().SetStatQueueProfile(sgs, true); err != nil {
			return fmt.Errorf("error: <%s> when setting statQueue profile with tenant: <%s> and id: <%s>",
				err.Error(), tntID[0], tntID[1])
		}
		m.stats[utils.RQF]++
	}
	return
}

func (m *Migrator) migrateThresholdsProfileFiltersV2() (err error) {
	var ids []string
	ids, err = m.dmIN.DataManager().DataDB().GetKeysForPrefix(utils.ThresholdProfilePrefix)
	if err != nil {
		return fmt.Errorf("error: <%s> when getting threshold profile IDs", err)
	}
	for _, id := range ids {
		tntID := strings.SplitN(strings.TrimPrefix(id, utils.ThresholdProfilePrefix), utils.InInFieldSep, 2)
		if len(tntID) < 2 {
			return fmt.Errorf("Invalid key <%s> when migrating filter for thresholdProfile", id)
		}
		ths, err := m.dmIN.DataManager().GetThresholdProfile(tntID[0], tntID[1], false, false, utils.NonTransactional)
		if err != nil {
			return fmt.Errorf("error: <%s> when getting threshold profile with tenant: <%s> and id: <%s>",
				err.Error(), tntID[0], tntID[1])
		}
		if ths == nil || m.dryRun {
			continue
		}
		for i, fl := range ths.FilterIDs {
			ths.FilterIDs[i] = migrateInlineFilterV2(fl)
		}
		if err := m.dmOut.DataManager().SetThresholdProfile(ths, true); err != nil {
			return fmt.Errorf("error: <%s> when setting threshold profile with tenant: <%s> and id: <%s>",
				err.Error(), tntID[0], tntID[1])
		}
		m.stats[utils.RQF]++
	}
	return
}

func (m *Migrator) migrateSupplierProfileFiltersV2() (err error) {
	for {
		var spp *SupplierProfile
		spp, err = m.dmIN.getSupplier()
		if err == utils.ErrNoMoreData {
			err = nil
			break
		}
		if err != nil {
			return err
		}
		if spp == nil || m.dryRun {
			continue
		}
		for i, fl := range spp.FilterIDs {
			spp.FilterIDs[i] = migrateInlineFilterV2(fl)
		}
		if err := m.dmOut.setSupplier(spp); err != nil {
			return fmt.Errorf("error: <%s> when setting supplier profile with tenant: <%s> and id: <%s>",
				err.Error(), spp.Tenant, spp.ID)
		}
		m.stats[utils.RQF]++
	}
	return
}

func (m *Migrator) migrateAttributeProfileFiltersV2() (err error) {
	var ids []string
	ids, err = m.dmIN.DataManager().DataDB().GetKeysForPrefix(utils.AttributeProfilePrefix)
	if err != nil {
		return fmt.Errorf("error: <%s> when getting attribute profile IDs", err)
	}
	for _, id := range ids {
		tntID := strings.SplitN(strings.TrimPrefix(id, utils.AttributeProfilePrefix), utils.InInFieldSep, 2)
		if len(tntID) < 2 {
			return fmt.Errorf("Invalid key <%s> when migrating filter for attributeProfile", id)
		}
		attrPrf, err := m.dmIN.DataManager().GetAttributeProfile(tntID[0], tntID[1], false, false, utils.NonTransactional)
		if err != nil {
			return fmt.Errorf("error: <%s> when getting attribute profile with tenant: <%s> and id: <%s>",
				err.Error(), tntID[0], tntID[1])
		}
		if attrPrf == nil || m.dryRun {
			continue
		}
		for i, fl := range attrPrf.FilterIDs {
			attrPrf.FilterIDs[i] = migrateInlineFilterV2(fl)
		}
		for i, attr := range attrPrf.Attributes {
			for j, fl := range attr.FilterIDs {
				attrPrf.Attributes[i].FilterIDs[j] = migrateInlineFilterV2(fl)
			}
		}
		if err := m.dmOut.DataManager().SetAttributeProfile(attrPrf, true); err != nil {
			return fmt.Errorf("error: <%s> when setting attribute profile with tenant: <%s> and id: <%s>",
				err.Error(), tntID[0], tntID[1])
		}
		m.stats[utils.RQF]++
	}
	return
}

func (m *Migrator) migrateChargerProfileFiltersV2() (err error) {
	var ids []string
	ids, err = m.dmIN.DataManager().DataDB().GetKeysForPrefix(utils.ChargerProfilePrefix)
	if err != nil {
		return fmt.Errorf("error: <%s> when getting charger profile IDs", err)
	}
	for _, id := range ids {
		tntID := strings.SplitN(strings.TrimPrefix(id, utils.ChargerProfilePrefix), utils.InInFieldSep, 2)
		if len(tntID) < 2 {
			return fmt.Errorf("Invalid key <%s> when migrating filter for chargerProfile", id)
		}
		cpp, err := m.dmIN.DataManager().GetChargerProfile(tntID[0], tntID[1], false, false, utils.NonTransactional)
		if err != nil {
			return fmt.Errorf("error: <%s> when getting charger profile with tenant: <%s> and id: <%s>",
				err.Error(), tntID[0], tntID[1])
		}
		if cpp == nil || m.dryRun {
			continue
		}
		for i, fl := range cpp.FilterIDs {
			cpp.FilterIDs[i] = migrateInlineFilterV2(fl)
		}
		if err := m.dmOut.DataManager().SetChargerProfile(cpp, true); err != nil {
			return fmt.Errorf("error: <%s> when setting charger profile with tenant: <%s> and id: <%s>",
				err.Error(), tntID[0], tntID[1])
		}
		m.stats[utils.RQF]++
	}
	return
}

func (m *Migrator) migrateDispatcherProfileFiltersV2() (err error) {
	var ids []string
	ids, err = m.dmIN.DataManager().DataDB().GetKeysForPrefix(utils.DispatcherProfilePrefix)
	if err != nil {
		return fmt.Errorf("error: <%s> when getting dispatcher profile IDs", err)
	}
	for _, id := range ids {
		tntID := strings.SplitN(strings.TrimPrefix(id, utils.DispatcherProfilePrefix), utils.InInFieldSep, 2)
		if len(tntID) < 2 {
			return fmt.Errorf("Invalid key <%s> when migrating filter for dispatcherProfile", id)
		}
		dpp, err := m.dmIN.DataManager().GetDispatcherProfile(tntID[0], tntID[1], false, false, utils.NonTransactional)
		if err != nil {
			return fmt.Errorf("error: <%s> when getting dispatcher profile with tenant: <%s> and id: <%s>",
				err.Error(), tntID[0], tntID[1])
		}
		if dpp == nil || m.dryRun {
			continue
		}
		for i, fl := range dpp.FilterIDs {
			dpp.FilterIDs[i] = migrateInlineFilterV2(fl)
		}
		if err := m.dmOut.DataManager().SetDispatcherProfile(dpp, true); err != nil {
			return fmt.Errorf("error: <%s> when setting dispatcher profile with tenant: <%s> and id: <%s>",
				err.Error(), tntID[0], tntID[1])
		}
		m.stats[utils.RQF]++
	}
	return
}

type v1Filter struct {
	Tenant             string
	ID                 string
	Rules              []*v1FilterRule
	ActivationInterval *utils.ActivationInterval
}

type v1FilterRule struct {
	Type      string            // Filter type (*string, *timing, *rsr_filters, *stats, *lt, *lte, *gt, *gte)
	FieldName string            // Name of the field providing us the Values to check (used in case of some )
	Values    []string          // Filter definition
	rsrFields config.RSRParsers // Cache here the RSRFilter Values
	negative  *bool
}
