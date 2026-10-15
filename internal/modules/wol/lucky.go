package wol

import (
	"encoding/json"
	"fmt"

	"github.com/wangwanjie/gately/internal/crud"
	"github.com/wangwanjie/gately/internal/luckyimport"
)

type luckyDevice struct {
	DeviceName          string
	MacList             []string
	BroadcastIPs        []string
	Port                int
	Relay               bool
	Repeat              int
	IOT_DianDeng_Enable bool
	IOT_Bemfa_Enable    bool
}

// ImportLucky 实现 luckyimport.Importer。
func (m *Module) ImportLucky(doc luckyimport.Document, dryRun bool) (*luckyimport.Report, error) {
	raw, ok := doc["WOLDeviceList"]
	if !ok {
		return nil, nil
	}
	var src []luckyDevice
	if err := json.Unmarshal(raw, &src); err != nil {
		return nil, err
	}
	rep := &luckyimport.Report{Module: ID, Title: m.Title(), Found: len(src)}
	var devices []Device
	for _, ld := range src {
		d := Device{Name: ld.DeviceName, MACs: ld.MacList, BroadcastIPs: ld.BroadcastIPs, Port: ld.Port, Repeat: ld.Repeat}
		if ld.IOT_DianDeng_Enable || ld.IOT_Bemfa_Enable {
			rep.Warnings = append(rep.Warnings, ld.DeviceName+": 点灯科技/巴法云语音接入暂不支持，已忽略")
		}
		if err := d.Validate(); err != nil {
			rep.Warnings = append(rep.Warnings, fmt.Sprintf("%s: 跳过，%v", ld.DeviceName, err))
			continue
		}
		devices = append(devices, d)
	}
	rep.Imported = len(devices)
	if dryRun {
		return rep, nil
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	old := m.cfg.Devices
	for _, d := range devices {
		d.Key = crud.NewKey()
		m.cfg.Devices = append(m.cfg.Devices, d)
	}
	if err := m.saveLocked(); err != nil {
		m.cfg.Devices = old
		return nil, err
	}
	return rep, nil
}
