package globalentrylist

import (
	"path"

	"github.com/assetto-corsa-web/accweb/internal/pkg/event"
	"github.com/assetto-corsa-web/accweb/internal/pkg/helper"
	"github.com/assetto-corsa-web/accweb/internal/pkg/instance"
	"github.com/assetto-corsa-web/accweb/internal/pkg/server_manager"
	"github.com/sirupsen/logrus"
)

var sM *server_manager.Service

func Register(sm *server_manager.Service) {
	sM = sm
	event.Register(handleEvent)
}

func handleEvent(data event.Eventer) {
	switch ev := data.(type) {
	case event.EventInstanceBeforeStart:
		i, err := sM.GetServerByID(ev.InstanceId)
		if err != nil {
			logrus.WithError(err).Error("instance not found")
			return
		}

		if !i.Cfg.Settings.EnableGlobalAdmin && !i.Cfg.Settings.EnableGlobalBan {
			return
		}

		list := i.AccCfg.Entrylist
		list.ForceEntryList = 1

		if i.Cfg.Settings.EnableGlobalAdmin {
			admins, err := sM.LoadGlobalEntry(server_manager.GlobalEntryContextAdmin)
			if err != nil {
				logrus.WithError(err).Error("failed to load global admin list")
				return
			}

			t := 1

			for _, entry := range admins {
				list.Entries = append(list.Entries, instance.EntrySettings{
					Drivers: []instance.DriverSettings{
						{
							PlayerID:  entry.PlayerId,
							FirstName: &entry.PlayerName,
						},
					},
					IsServerAdmin: &t,
				})
			}
		}

		if i.Cfg.Settings.EnableGlobalBan {
			bans, err := sM.LoadGlobalEntry(server_manager.GlobalEntryContextBan)
			if err != nil {
				logrus.WithError(err).Error("failed to load global admin list")
				return
			}

			carModel := 9999

			for _, entry := range bans {
				list.Entries = append(list.Entries, instance.EntrySettings{
					Drivers: []instance.DriverSettings{
						{
							PlayerID:  entry.PlayerId,
							FirstName: &entry.PlayerName,
						},
					},
					ForcedCarModel: &carModel,
				})
			}
		}

		helper.SaveToPath(path.Join(i.Path, "cfg"), "entrylist.json", list)
	}
}
