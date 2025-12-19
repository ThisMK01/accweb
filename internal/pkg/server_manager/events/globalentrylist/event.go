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

		if !i.Cfg.Settings.EnableGlobalEntrylist && !i.Cfg.Settings.EnableGlobalBanlist {
			return
		}

		list := i.AccCfg.Entrylist

		if i.Cfg.Settings.EnableGlobalEntrylist {
			prepareEntrylist(&list)
		}

		if i.Cfg.Settings.EnableGlobalBanlist {
			prepareBanList(&list)
		}

		helper.SaveToPath(path.Join(i.Path, "cfg"), "entrylist.json", list)
	}
}

func prepareEntrylist(list *instance.EntrylistJson) {
	var data instance.AccwebGlobalEntrylistJson
	err := sM.LoadGlobalEntry(server_manager.GlobalListCtxEntry, &data)
	if err != nil {
		logrus.WithError(err).Error("failed to load global entry list")
		return
	}

	if !data.Enabled {
		return
	}

	for _, entry := range data.Entries {
		list.Entries = append(list.Entries, entry)
	}

	list.ForceEntryList = 1
}

func prepareBanList(list *instance.EntrylistJson) {
	var data instance.AccwebGlobalBanlistJson
	if err := sM.LoadGlobalEntry(server_manager.GlobalEntryCtxBan, &data); err != nil {
		logrus.WithError(err).Error("failed to load global ban list")
		return
	}

	if !data.Enabled {
		return
	}

	carModel := 9999

	for _, entry := range data.Entries {
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

	list.ForceEntryList = 1
}
