package synapse

import (
	"github.com/samuel/go-zookeeper/zk"
	"github.com/n0rad/go-erlog/errs"
	"sync"
	"github.com/blablacar/go-nerve/nerve"
	"time"
)

type WatcherZookeeper struct {
	WatcherCommon
	Hosts          []string
	Path           string
	TimeoutInMilli int

	connection     *zk.Conn
}

func NewWatcherZookeeper() *WatcherZookeeper {
	return &WatcherZookeeper{
		TimeoutInMilli: 2000,
	}
}

func (w *WatcherZookeeper) Init() error {
	if err := w.CommonInit(); err != nil {
		return errs.WithEF(err, w.fields, "Failed to init discovery")
	}
	//zd.zkConnection = nil
	return nil
}
//
//func (zd *WatcherZookeeper) Connect() (zk.State, error) {
//	if zd.zkConnection != nil {
//		state := zd.zkConnection.State()
//		switch state {
//		case zk.StateUnknown, zk.StateConnectedReadOnly, zk.StateExpired, zk.StateAuthFailed, zk.StateConnecting:
//			{
//				//Disconnect, and let Reconnection happen
//				log.Warn("Zookeeper Connection is in BAD State [", state, "] Reconnect")
//				zd.zkConnection.Close()
//			}
//		case zk.StateConnected, zk.StateHasSession:
//			{
//				log.Debug("Zookeeper Connection established (", state, "), nothing to do.")
//				return state, nil
//			}
//		case zk.StateDisconnected:
//			{
//				log.Info("Reporter Connection is Disconnected -> Reconnection")
//			}
//		}
//	}
//	var err error
//	zd.zkConnection, zd.ZKConnEvent, err = zk.Connect(zd.Hosts, time.Second)
//	if err != nil {
//		zd.zkConnection = nil
//		log.Warn("Unable to Connect to ZooKeeper (", err, ")")
//		return zk.StateDisconnected, err
//	}
//	state := zd.zkConnection.State()
//	return state, nil
//}
//
//func (zd *WatcherZookeeper) watchLoop(snapshots chan []string, error_chan chan error, watchChildsSignal chan bool) {
//	defer zd.waitGroup.Done()
//	for {
//		snapshot, _, events, err := zd.zkConnection.ChildrenW(zd.Path)
//		if err != nil {
//			error_chan <- err
//			watchChildsSignal <- true
//			return
//		}
//		snapshots <- snapshot
//		var event zk.Event
//		select {
//		case event = <-events:
//			if event.State == zk.StateDisconnected {
//				error_chan <- errors.New("ZK Connection is closed")
//				watchChildsSignal <- true
//				return
//			}
//		case signal := <-zd.destroySignal:
//			if signal {
//				log.Info("Kill signal receive in Zookeeper Discovery Watch")
//			} else {
//				log.Warn("Kill signal receive in Zookeeper Discovery Watch, but ?? False ??")
//			}
//			return
//		}
//		if event.Err != nil {
//			log.WithError(event.Err).Warn("Error In Zookeeper Discovery watch")
//			error_chan <- event.Err
//			watchChildsSignal <- true
//			return
//		}
//	}
//	return
//}
//
//func (zd *WatcherZookeeper) WatchForChildren(watchChildsSignal chan bool) (chan []string, chan error) {
//	snapshots := make(chan []string)
//	error_chan := make(chan error)
//	zd.waitGroup.Add(1)
//	go zd.watchLoop(snapshots, error_chan, watchChildsSignal)
//	return snapshots, error_chan
//}
//
//func (zd *WatcherZookeeper) addNewDicoveredHost(hostList *[]DiscoveredHost, host string) error {
//	data, _, err := zd.zkConnection.Get(zd.Path + "/" + host)
//	if err != nil {
//		log.WithError(err).Warn("Unable to get data info for node [", zd.Path +"/"+host, "]")
//		return err
//	}
//	var discoveredHost DiscoveredHost
//	// Trying to convert the content of the ZNode Data (theoriticaly in JSON) into a configuration object
//	err = json.Unmarshal(data, &discoveredHost)
//	if err != nil {
//		// If there is an error in decoding the JSON entry into configuration object, print the error and continue
//		log.WithError(err).Warn("Unable to Parse JSON data for node [", zd.Path +"/"+host, "]")
//		return err
//	} else {
//		discoveredHost.ZKHostName = host
//		*hostList = append(*hostList, discoveredHost)
//	}
//	return nil
//}
//
//func (zd *WatcherZookeeper) updateDiscoveredHosts(HostList []string) error {
//	if len(HostList) == 0 {
//		if len(zd.Hosts) > 0 {
//			//We can empty the Node List
//			zd.Hosts = nil
//			zd.serviceModified <- true
//		}
//	} else {
//		var newHost []DiscoveredHost
//		for _, host := range HostList {
//			zd.addNewDicoveredHost(&newHost, host)
//		}
//		zd.Hosts = newHost
//		zd.serviceModified <- true
//	}
//	return nil
//}
//
//func (zd *WatcherZookeeper) InitializeDiscovery(updateHostSignal chan bool, watchChildsSignal chan bool) error {
//	//Test Connection to ZooKeeper
//	state, err := zd.Connect() //internally the connection is maintained
//	log.Debug("ZK Connection State After Connect [", state, "]")
//	if err != nil {
//		log.Warn("Unable to Discover... Connection to Zookeeper Fail")
//		return err
//	}
//	//Put a time to wait for connection to be established
//	time.Sleep(3 * time.Second)
//	state = zd.zkConnection.State()
//	if state == zk.StateHasSession {
//		exists, _, _ := zd.zkConnection.Exists(zd.Path)
//		if exists {
//			//First get All Childs
//			children, stats, err := zd.zkConnection.Children(zd.Path)
//			if err != nil {
//				log.WithError(err).Warn("Zookeeper Discovery: First Check of childs for [", zd.Path, "] failed, exiting")
//				return err
//			}
//			if stats.NumChildren > 0 {
//				err = zd.updateDiscoveredHosts(children)
//				if err != nil {
//					log.WithError(err).Warn("Zookeeper Discovery: Failed to grap all children info of [", zd.Path, "]")
//				}
//			}
//			//Second create a subscription to any change on the path
//			snapshots, errors := zd.WatchForChildren(watchChildsSignal)
//			zd.waitGroup.Add(1)
//			go zd.watchSignals(snapshots, errors, updateHostSignal)
//		}
//	}
//	return nil
//}
//
//func (zd *WatcherZookeeper) watchSignals(snapshots chan []string, errors chan error, updateHostSignal chan bool) {
//	defer zd.waitGroup.Done()
//	for {
//		select {
//		case snapshot := <-snapshots:
//			//Here, we need to update Hosts lists
//			log.Debug("Snaphost received, update Discovered Hosts List")
//			zd.updateDiscoveredHosts(snapshot)
//		case err := <-errors:
//			//Will stop the discovery process
//			//Perhaps need a better error management
//			//But until a fully tested case, will exit now!
//			log.WithError(err).Warn("Zookeeper Discovery has an error, Exiting")
//			updateHostSignal <- true
//			return
//		case signal := <-zd.destroySignal:
//			//stopping the loop, time to leave!
//			if signal {
//				log.Info("Kill signal receive in Zookeeper Discovery")
//			} else {
//				log.Warn("Kill signal receive in Zookeeper Discovery, but ?? False ??")
//			}
//			return
//		}
//	}
//}
//
func (w *WatcherZookeeper) Run(stop <-chan bool, doneWaiter *sync.WaitGroup, events chan<- []nerve.Report) {
	doneWaiter.Add(1)
	defer doneWaiter.Done()

	for {
		_, _, err := zk.Connect(w.Hosts, time.Duration(w.TimeoutInMilli) * time.Millisecond)
		if err != nil {

		}


		select {
		case <- stop:
			// TODO stop
			return
		}
	}
}
//
////func (zd *WatcherZookeeper) Run(stop <-chan bool) error {
////	updateHostRoutine := make(chan bool, 1)
////	watchChildsRoutine := make(chan bool, 1)
////Loop:
////	for {
////		err := zd.InitializeDiscovery(updateHostRoutine, watchChildsRoutine)
////		if err != nil {
////			log.Warn("Initialization Failed")
////		}
////	StopLoop:
////		for {
////			select {
////			case <-stop:
////				log.Warn("Zookeeper Discovery, stopSignal Received")
////				break Loop
////			case <-updateHostRoutine:
////				log.Warn("Update Host Routine down - Restarting the whole Discovery Process")
////				zd.destroyOneLoop()
////				break StopLoop
////			case <-watchChildsRoutine:
////				log.Warn("Watch Childs Routine down - Restarting the whole Discovery Process")
////				zd.Destroy()
////				break StopLoop
////			case ev := <-zd.ZKConnEvent:
////				if ev.Err != nil {
////					log.WithError(ev.Err).Warn("Connection problem... Restarting it")
////					err := zd.destroyTwoLoop()
////					if err != nil {
////						log.WithError(err).Warn("Problem closing all connections")
////					}
////					break StopLoop
////				} else {
////					log.Debug("Connection Event [", ev.Type, "][", ev.State, "]")
////				}
////			}
////		}
////		time.Sleep(500 * time.Millisecond)
////	}
////	return zd.destroyTwoLoop()
////}
