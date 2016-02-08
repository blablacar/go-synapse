package discovery

import (
	log "github.com/Sirupsen/logrus"
	"github.com/samuel/go-zookeeper/zk"
	"encoding/json"
	"time"
	"sync"
)

const DISCOVERY_ZOOKEEPER_TYPE string = "ZOOKEEPER"

type zookeeperDiscovery struct {
	Discovery
	ZKHosts []string
	ZKPath string
	ZKConnection *zk.Conn
	ZKConnEvent <-chan zk.Event
	destroySignal chan bool
	waitGroup sync.WaitGroup
}


func(zd *zookeeperDiscovery) Initialize() {
	zd.Type = DISCOVERY_ZOOKEEPER_TYPE
	zd.ZKConnection = nil
	zd.destroySignal = make(chan bool,2)
}

func(zd *zookeeperDiscovery) SetZKConfiguration(ZKHosts []string, ZKPath string) {
	if len(ZKHosts) > 0 {
		zd.ZKHosts = ZKHosts
	}
	if ZKPath != "" {
		zd.ZKPath = ZKPath
	}
}

func(zd *zookeeperDiscovery) Connect() (zk.State, error) {
	if zd.ZKConnection != nil {
		state := zd.ZKConnection.State()
		switch state {
			case zk.StateUnknown,zk.StateConnectedReadOnly,zk.StateExpired,zk.StateAuthFailed,zk.StateConnecting: {
				//Disconnect, and let Reconnection happen
				log.Warn("Zookeeper Connection is in BAD State [",state,"] Reconnect")
				zd.ZKConnection.Close()
			}
			case zk.StateConnected, zk.StateHasSession: {
				log.Debug("Zookeeper Connection established (",state,"), nothing to do.")
				return state, nil
			}
			case zk.StateDisconnected: {
				log.Info("Reporter Connection is Disconnected -> Reconnection")
			}
		}
	}
	var err error
	zd.ZKConnection, zd.ZKConnEvent, err = zk.Connect(zd.ZKHosts, time.Second)
	if err != nil {
		zd.ZKConnection = nil
		log.Warn("Unable to Connect to ZooKeeper (",err,")")
		return zk.StateDisconnected, err
	}
	state := zd.ZKConnection.State()
	return state, nil
}

func(zd *zookeeperDiscovery) WatchForChildren(watchChildsSignal chan bool) (chan []string, chan error) {
	defer zd.waitGroup.Done()
	snapshots := make(chan []string)
	errors := make(chan error)
	go func() {
		snapshot, _, events, err := zd.ZKConnection.ChildrenW(zd.ZKPath)
		if err != nil {
			errors <- err
			watchChildsSignal <- true
			return
		}
		for {
			var event zk.Event
			select {
			case event = <-events:
				snapshots <-snapshot
				log.Debug("Zookeeper Discovery: New Event Receive from [",zd.ZKPath,"]")
			case signal := <-zd.destroySignal:
				if signal {
					log.Info("Kill signal receive in Zookeeper Discovery Watch")
				}else {
					log.Warn("Kill signal receive in Zookeeper Discovery Watch, but ?? False ??")
				}
				watchChildsSignal <- true
				return
			}
			if event.Err != nil {
				log.WithError(event.Err).Warn("Error In Zookeeper Discovery watch")
				errors <-event.Err
				watchChildsSignal <- true
				return
			}
		}
	}()
	return snapshots, errors
}

func(zd *zookeeperDiscovery) updateDiscoveredHosts(HostList []string) error {
	var newDiscoveredHost []DiscoveredHost
	for _, host := range HostList {
		data, _, err := zd.ZKConnection.Get(zd.ZKPath+"/"+host)
		if err != nil {
			log.WithError(err).Warn("Unable to get data info for node [",zd.ZKPath+"/"+host,"]")
			return err
		}
		var discoveredHost DiscoveredHost
		// Trying to convert the content of the ZNode Data (theoriticaly in JSON) into a configuration object
		err = json.Unmarshal(data, &discoveredHost)
		if err != nil {
			// If there is an error in decoding the JSON entry into configuration object, print the error and continue
			log.WithError(err).Warn("Unable to Parse JSON data for node [",zd.ZKPath+"/"+host,"]")
		}else {
			newDiscoveredHost = append(newDiscoveredHost,discoveredHost)
		}
	}
	zd.Hosts = newDiscoveredHost
	return nil
}

func(zd *zookeeperDiscovery) InitializeDiscovery(updateHostSignal chan bool, watchChildsSignal chan bool) error {
	zd.waitGroup.Add(1)
	defer zd.waitGroup.Done()
	//Test Connection to ZooKeeper
	state, err := zd.Connect() //internally the connection is maintained
	if err != nil {
		log.Warn("Unable to Discover... Connection to Zookeeper Fail")
		return err
	}
	if state == zk.StateHasSession {
		exists, _, _ := zd.ZKConnection.Exists(zd.ZKPath)
		if exists {
		//First get All Childs
			children, stats, err := zd.ZKConnection.Children(zd.ZKPath)
			if err != nil {
				log.WithError(err).Warn("Zookeeper Discovery: First Check of childs for [",zd.ZKPath,"] failed, exiting")
				return err
			}
			if stats.NumChildren > 0 {
				err = zd.updateDiscoveredHosts(children)
				if err != nil {
					log.WithError(err).Warn("Zookeeper Discovery: Failed to grap all children info of [",zd.ZKPath,"]")
				}
			}
			zd.waitGroup.Add(1)
		//Second create a subscription to any change on the path
			snapshots, errors := zd.WatchForChildren(watchChildsSignal)
			go func() {
				for {
					select {
					case snapshot := <-snapshots:
						//Here, we need to update Hosts lists
						zd.updateDiscoveredHosts(snapshot)
					case err := <-errors:
						//Will stop the discovery process
						//Perhaps need a better error management
						//But until a fully tested case, will exit now!
						log.WithError(err).Warn("Zookeeper Discovery has an error, Exiting")
						updateHostSignal <- true
						return
					case signal := <-zd.destroySignal:
						//stopping the loop, time to leave!
						if signal {
							log.Info("Kill signal receive in Zookeeper Discovery")
						}else {
							log.Warn("Kill signal receive in Zookeeper Discovery, but ?? False ??")
						}
						updateHostSignal <- true
						return
					}
				}
			}()
		}
	}
	return nil
}

func(zd *zookeeperDiscovery) Run(stop <-chan bool) error {
	updateHostRoutine := make(chan bool,1)
	watchChildsRoutine := make(chan bool,1)
	Loop:
	for {
		err := zd.InitializeDiscovery(updateHostRoutine,watchChildsRoutine)
		if err != nil {
			log.Warn("Initialization Failed")
		}
		StopLoop:
		for {
			select {
				case <-stop:
					log.Warn("Zookeeper Discovery, stopSignal Received")
					break Loop
				case <-updateHostRoutine:
					log.Warn("Update Host Routine down - Restarting the whole Discovery Process")
					zd.destroyOneLoop()
					break StopLoop
				case <-watchChildsRoutine:
					log.Warn("Watch Childs Routine down - Restarting the whole Discovery Process")
					zd.Destroy()
					break StopLoop
				case ev := <-zd.ZKConnEvent:
					log.WithError(ev.Err).Warn("Connection problem... Restarting it")
					err := zd.destroyTwoLoop()
					if err != nil {
						log.WithError(err).Warn("Problem closing all connections")
					}
					break StopLoop
				default:
					time.Sleep(500 * time.Millisecond)
			}
		}
		time.Sleep(500 * time.Millisecond)
	}
	return zd.destroyTwoLoop()
}

func(zd *zookeeperDiscovery) destroyTwoLoop() error {
	//Send 2 signals to kill the discovery loops
	zd.destroySignal <- true
	zd.destroySignal <- true
	return zd.Destroy()
}

func(zd *zookeeperDiscovery) destroyOneLoop() error {
	zd.destroySignal <- true
	return zd.Destroy()
}

func(zd *zookeeperDiscovery) Destroy() error {
	//Wait for all thread to terminate
	zd.waitGroup.Wait()
	//Close properly the connection to Zookeeper
	if zd.ZKConnection != nil {
		zd.ZKConnection.Close()
		zd.ZKConnection = nil
	}
	return nil
}

func(zd *zookeeperDiscovery) GetType() string {
	return zd.Type
}
