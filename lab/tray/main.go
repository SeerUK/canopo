package main

import (
	"fmt"
	"os"
	"regexp"
	"time"

	"github.com/godbus/dbus"
	"github.com/godbus/dbus/introspect"
	"github.com/godbus/dbus/prop"
)

const (
	NotifierHostService            = "org.kde.StatusNotifierHost-%d"
	NotifierItemPath               = "/StatusNotifierItem"
	NotifierItemInterface          = "org.kde.StatusNotifierItem"
	NotifierItemService            = "org.kde.StatusNotifierItem-%d-%d"
	NotifierWatcherPath            = "/StatusNotifierWatcher"
	NotifierWatcherService         = "org.kde.StatusNotifierWatcher"
	NotifierWatcherRegisterHost    = "RegisterStatusNotifierHost"
	NotifierIntrospectionInterface = "org.freedesktop.DBus.Introspectable"
	PropertiesInterface            = "org.freedesktop.DBus.Properties"
)

func senderAndPath(serviceName string, sender dbus.Sender) (string, dbus.ObjectPath) {
	if regexp.MustCompile("^(/\\w+)+$").MatchString(serviceName) {
		return string(sender), dbus.ObjectPath(serviceName)
	} else {
		return string(serviceName), dbus.ObjectPath(NotifierItemPath)
	}
}

func main() {
	conn, err := dbus.SessionBus()
	if err != nil {
		panic(err)
	}

	// Add introspection, for something?
	conn.Export(introspect.Introspectable(IntrospectXML), NotifierWatcherPath, NotifierIntrospectionInterface)

	// Add watcher methods.
	conn.ExportMethodTable(
		map[string]interface{}{
			"RegisterStatusNotifierItem": func(service string, sender dbus.Sender) *dbus.Error {
				fmt.Println("ITEM REGISTERING?")
				fmt.Println(service)
				fmt.Printf("%+v\n", sender)

				method := fmt.Sprintf("%s.Get", PropertiesInterface)
				fmt.Println(method)

				es, ep := senderAndPath(service, sender)
				fmt.Println(es, ep)

				itemObj := conn.Object(es, ep)
				call := itemObj.Call(method, dbus.Flags(64), NotifierItemInterface, "Title")
				if call.Err != nil {
					return dbus.NewError(call.Err.Error(), nil)
				}

				title, _ := itemObj.GetProperty(fmt.Sprintf("%s.Title", NotifierItemInterface))
				status, _ := itemObj.GetProperty(fmt.Sprintf("%s.Status", NotifierItemInterface))
				windowID, _ := itemObj.GetProperty(fmt.Sprintf("%s.WindowId", NotifierItemInterface))
				iconName, _ := itemObj.GetProperty(fmt.Sprintf("%s.IconName", NotifierItemInterface))
				//iconPixmap, _ := itemObj.GetProperty(fmt.Sprintf("%s.IconPixmap", NotifierItemInterface))
				overlayIconName, _ := itemObj.GetProperty(fmt.Sprintf("%s.OverlayIconName", NotifierItemInterface))
				overlayIconPixmap, _ := itemObj.GetProperty(fmt.Sprintf("%s.OverlayIconPixmap", NotifierItemInterface))
				attentionIconName, _ := itemObj.GetProperty(fmt.Sprintf("%s.AttentionIconName", NotifierItemInterface))
				attentionIconPixmap, _ := itemObj.GetProperty(fmt.Sprintf("%s.AttentionIconPixmap", NotifierItemInterface))
				attentionMovieName, _ := itemObj.GetProperty(fmt.Sprintf("%s.AttentionMovieName", NotifierItemInterface))
				tooltip, _ := itemObj.GetProperty(fmt.Sprintf("%s.ToolTip", NotifierItemInterface))
				itemIsMenu, _ := itemObj.GetProperty(fmt.Sprintf("%s.ItemIsMenu", NotifierItemInterface))
				menu, _ := itemObj.GetProperty(fmt.Sprintf("%s.Menu", NotifierItemInterface))

				fmt.Println(title.String())
				fmt.Println(status.String())
				if windowID.Value() != nil {
					fmt.Println("windowID", windowID.String())
				}
				if iconName.Value() != nil {
					fmt.Println("iconName", iconName.String())
				}
				//if iconPixmap.Value() != nil {
				//	fmt.Println("iconPixmap", iconPixmap.String())
				//}
				if overlayIconName.Value() != nil {
					fmt.Println("overlayIconName", overlayIconName.String())
				}
				if overlayIconPixmap.Value() != nil {
					fmt.Println("overlayIconPixmap", overlayIconPixmap.String())
				}
				if attentionIconName.Value() != nil {
					fmt.Println("attentionIconName", attentionIconName.String())
				}
				if attentionIconPixmap.Value() != nil {
					fmt.Println("attentionIconPixmap", attentionIconPixmap.String())
				}
				if attentionMovieName.Value() != nil {
					fmt.Println("attentionMovieName", attentionMovieName.String())
				}
				if tooltip.Value() != nil {
					fmt.Println("tooltip", tooltip.String())
				}
				if itemIsMenu.Value() != nil {
					fmt.Println("itemIsMenu", itemIsMenu.String())
				}
				if menu.Value() != nil {
					fmt.Println("menu", menu.String())
				}

				return nil
			},
			"RegisterStatusNotifierHost": func(service string, sender dbus.Sender) *dbus.Error {
				fmt.Println("HOST REGISTERING?")
				return nil
			},
		},
		NotifierWatcherPath,
		NotifierWatcherService,
	)

	prop.New(conn, NotifierWatcherPath, map[string]map[string]*prop.Prop{
		NotifierWatcherService: {
			"IsStatusNotifierHostRegistered": {true, false, prop.EmitTrue, nil},
			"ProtocolVersion":                {0, false, prop.EmitTrue, nil},
			"RegisteredStatusItems":          {[]string{}, false, prop.EmitTrue, nil},
		},
	})

	// Request the name of the watcher service, and do not queue up to wait to become the primary
	// owner of the name. Messages get sent to this service if it is the primary.
	reply, err := conn.RequestName(NotifierWatcherService, dbus.NameFlagDoNotQueue)
	if err != nil {
		panic(err)
	}

	// If we aren't the primary, we want to bail.
	if reply != dbus.RequestNameReplyPrimaryOwner {
		panic(fmt.Errorf("service %s already taken", NotifierWatcherService))
	}

	host := fmt.Sprintf(NotifierHostService, os.Getpid())

	// Request the name of the host service. The host must only be present on the bus, as primary.
	// After this, the host should be sent items by the watcher. The presence of both the watcher
	// and the host are what allows items to be retrieved (I think?)
	reply, err = conn.RequestName(host, dbus.NameFlagDoNotQueue)
	if err != nil {
		panic(err)
	}

	// If this host is already registered (but... how?), then bail.
	if reply != dbus.RequestNameReplyPrimaryOwner {
		panic(fmt.Errorf("service %s already taken", host))
	}

	go func() {
		signals := make(chan *dbus.Signal, 100)
		conn.Signal(signals)

		for signal := range signals {
			fmt.Printf("SIGNAL: %+v\n", signal)
		}
	}()

	time.Sleep(5 * time.Hour)

	//hostObj := conn.Object(NotifierWatcherService, NotifierWatcherPath)
	//
	//// Register our host in the watcher.
	//call := hostObj.Call(NotifierWatcherRegisterHost, 0, host)
	//if call.Err != nil {
	//	panic(call.Err)
	//}
	//
	//fmt.Printf("%+v\n", call)
}
