package main

import (
	"time"
	"flag"
	"github.com/toolkits/file"
	"sync"
	"database/sql"
	_ "github.com/go-sql-driver/mysql"

	"encoding/json"
	"log"
	"net/http"
)

type DatabaseConfig struct {
	Addr       string `json:"addr"`
	Account    string `json:"account"`
	Password   string `json:"password"`
}

type GlobalConfig struct {
	Debug         bool             `json:"debug"`
	Hostname      string           `json:"hostname"`
	IP            string           `json:"ip"`
	Database      *DatabaseConfig  `json:"database"`
}

type InterfacesContent struct {
	Type int64						`json:"type"`
	Main int64						`json:"main"`
	Useip int64						`json:"useip"`
	Ip string						`json:"ip"`
	Dns string						`json:"dns"`
	Port string						`json:"port"`
}

type GroupsContent struct {
	Groupid string					`json:"groupid"`
}

type TemplatesContent struct {
	Templateid string				`json:"templateid"`
}

type InventoryContent struct {
	Macaddress_a string				`json:"macaddress_a"`
	Macaddress_b string				`json:"macaddress_b"`
}

type ParamsContent struct {
	Host string							`json:"host"`
	Interfaces [] *InterfacesContent	`json:"interfaces"`
	Groups [] *GroupsContent			`json:"groups"`
	Templates [] *TemplatesContent		`json:"templates"`
	Inventory *InventoryContent			`json:"inventory"`
}

type Counter struct {
	Jsonrpc string					`json:"jsonrpc"`
	Method string					`json:"method"`
	Params *ParamsContent			`json:"params"`
	Id int							`json:"id"`
	Auth string						`json:"auth"`
}

var (
	ConfigFile string
	config     *GlobalConfig
	lock       = new(sync.RWMutex)
)



func initDb() {
	// log.Println(config.Database.Addr)
	str := config.Database.Account + ":" + config.Database.Password + "@tcp(" + config.Database.Addr + ")/graph?charset=utf8"
	// log.Println(str)
	db, err := sql.Open("mysql", str)
	db.SetMaxOpenConns(2000)
	db.SetMaxIdleConns(1000)
	defer db.Close()

	if err != nil {
		log.Println("Oh noez, could not connect to database")
		return
	}
}

func readDb(endpointId int) {
	str := config.Database.Account + ":" + config.Database.Password + "@tcp(" + config.Database.Addr + ")/graph?charset=utf8"
	// log.Println(str)
	db, err := sql.Open("mysql", str)
	db.SetMaxOpenConns(2000)
	db.SetMaxIdleConns(1000)
	defer db.Close()

	if err != nil {
		log.Println("Oh noez, could not connect to database")
		return
	}
	// initDb()
	
	// Prepare statement for reading data
	stmtOut, err := db.Prepare("SELECT endpoint FROM graph.endpoint WHERE id = ?")
	if err != nil {
		log.Println(err.Error())
	}
	defer stmtOut.Close()

	var endpoint string // we "scan" the result in here

	err = stmtOut.QueryRow(endpointId).Scan(&endpoint) // WHERE id = endpointId
	if err != nil {
		log.Println(err.Error())
		return
	}
	log.Printf("The endpoint name of %d is: %s", endpointId, endpoint)
}

/**
 * @function name:	func writeDb(args map[string]string)
 * @description:	This function creates a host record in "endpoint" table.
 * @related issues:	OWL-085
 * @param:			args map[string]string
 * @return:			void
 * @author:			Don Hsieh
 * @since:			09/15/2015
 * @last modified: 	09/15/2015
 * @called by:		func apiParser(rw http.ResponseWriter, req *http.Request)
 */
func writeDb(args map[string]string) {
	str := config.Database.Account + ":" + config.Database.Password + "@tcp(" + config.Database.Addr + ")/graph?charset=utf8"
	db, err := sql.Open("mysql", str)
	db.SetMaxOpenConns(2000)
	db.SetMaxIdleConns(1000)
	defer db.Close()

	if err != nil {
		log.Println("Oh noez, could not connect to database")
		return
	}
	host := args["host"]
	t := time.Now()
	// timestamp := time.Now().Unix()
	timestamp := t.Unix()
	log.Println(timestamp)
	// now := time.Unix(timestamp, 0).Format(time.RFC3339)
	// now := time.Unix(timestamp, 0).Format("2006-01-02 15:04:05")
	now := t.Format("2006-01-02 15:04:05")
	
	// Prepare statement for inserting data
	stmtIns, err := db.Prepare("INSERT INTO graph.endpoint (endpoint,ts,t_create,t_modify) VALUES(?, ?, ?, ?)") // ? = placeholder
	if err != nil {
		log.Println(err.Error())
		return
	}
	defer stmtIns.Close() // Close the statement when we leave main() / the program terminates

	if result, err := stmtIns.Exec(host,timestamp,now,now); err==nil {
		if id, err := result.LastInsertId(); err==nil {
			log.Println("insert id : ", id);
		}
	}
}

/**
 * @function name:	func hostCreate(counter Counter)
 * @description:	This function gets host data for database insertion.
 * @related issues:	OWL-085
 * @param:			counter Counter
 * @return:			void
 * @author:			Don Hsieh
 * @since:			09/11/2015
 * @last modified: 	09/15/2015
 * @called by:		func apiParser(rw http.ResponseWriter, req *http.Request)
 */
func hostCreate(counter Counter) {
	host := counter.Params.Host
	ip := counter.Params.Interfaces[0].Ip
	port := counter.Params.Interfaces[0].Port
	groupId := counter.Params.Groups[0].Groupid
	templateId := counter.Params.Templates[0].Templateid
	macAddr := counter.Params.Inventory.Macaddress_a + counter.Params.Inventory.Macaddress_b

	args := map[string]string{
		"host": host, 
		"ip": ip,
		"port": port,
		"groupId": groupId,
		"templateId": templateId,
		"macAddr": macAddr,
	}
	log.Println(args)
	writeDb(args)
}

/**
 * @function name:	func apiParser(rw http.ResponseWriter, req *http.Request)
 * @description:	This function parses the method of API request.
 * @related issues:	OWL-085
 * @param:			rw http.ResponseWriter
 * @param:			req *http.Request
 * @return:			void
 * @author:			Don Hsieh
 * @since:			09/11/2015
 * @last modified: 	09/15/2015
 * @called by:		http.HandleFunc("/api", apiParser)
 *					 in func main()
 */
func apiParser(rw http.ResponseWriter, req *http.Request) {
	// log.Println(req.Body)
	decoder := json.NewDecoder(req.Body)
	var counter Counter
	err := decoder.Decode(&counter)
	if err != nil {
		log.Println(err.Error())
		return
	}
	method := counter.Method
	log.Println(method)
	if method == "host.create" {
		hostCreate(counter)
	}
}

/**
 * @function name:	func parseConfig(cfg string)
 * @description:	This function parses config file cfg.json.
 * @related issues:	OWL-085
 * @param:			cfg string
 * @return:			void
 * @author:			Don Hsieh
 * @since:			09/14/2015
 * @last modified: 	09/15/2015
 * @called by:		func main()
 */
func parseConfig(cfg string) {
	if !file.IsExist(cfg) {
		log.Fatalln("config file:", cfg, "is not existent. maybe you need `mv cfg.example.json cfg.json`")
	}
	ConfigFile = cfg
	configContent, err := file.ToTrimString(cfg)
	if err != nil {
		log.Fatalln("read config file:", cfg, "fail:", err)
	}

	var c GlobalConfig
	err = json.Unmarshal([]byte(configContent), &c)
	if err != nil {
		log.Fatalln("parse config file:", cfg, "fail:", err)
		return
	}
	lock.Lock()
	defer lock.Unlock()
	config = &c
	log.Println("read config file:", cfg, "successfully")
}

/**
 * @function name:	func main()
 * @description:	This function handles API requests.
 * @related issues:	OWL-085
 * @param:			void
 * @return:			void
 * @author:			Don Hsieh
 * @since:			09/09/2015
 * @last modified: 	09/15/2015
 * @called by:
 */
func main() {
	cfg := flag.String("c", "cfg.json", "configuration file")
	flag.Parse()
	parseConfig(*cfg)
	// initDb()
	// readDb(5)
	// log.Println(config.Database.Account)
	http.HandleFunc("/api", apiParser)
	log.Fatal(http.ListenAndServe(":80", nil))
}
