package main

import (
	"strconv"
	"github.com/bitly/go-simplejson"
	"bytes"
	// "io"
	"reflect"
	"os"
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

var (
	ConfigFile string
	config     *GlobalConfig
	lock       = new(sync.RWMutex)
)

func initDb() {
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
func writeDb(sqlcmd string, args []interface {}) {
	str := config.Database.Account + ":" + config.Database.Password + "@tcp(" + config.Database.Addr + ")/graph?charset=utf8"
	db, err := sql.Open("mysql", str)
	db.SetMaxOpenConns(2000)
	db.SetMaxIdleConns(1000)
	defer db.Close()

	if err != nil {
		log.Println("Oh noez, could not connect to database")
		return
	}
	
	// Prepare statement for inserting data
	// stmtIns, err := db.Prepare("INSERT INTO graph.endpoint (endpoint,ts,t_create,t_modify) VALUES(?, ?, ?, ?)") // ? = placeholder
	stmtIns, err := db.Prepare(sqlcmd)
	if err != nil {
		log.Println(err.Error())
		return
	}
	defer stmtIns.Close() // Close the statement when we leave main() / the program terminates

	// if result, err := stmtIns.Exec(host,timestamp,now,now); err==nil {
	if result, err := stmtIns.Exec(args); err==nil {
		if id, err := result.LastInsertId(); err==nil {
			log.Println("insert id :", id);
		} else {
			log.Println(err.Error())
		}
	} else {
		log.Println(err.Error())
	}
}

func RenderJson(w http.ResponseWriter, v interface{}) {
	bs, err := json.Marshal(v)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.Write(bs)
}

func RenderMsgJson(w http.ResponseWriter, msg string) {
	RenderJson(w, map[string]string{"msg": msg})
}

/**
 * @function name:	func hostCreate(nodes map[string]interface {}, rw http.ResponseWriter)
 * @description:	This function gets host data for database insertion.
 * @related issues:	OWL-086, OWL-085
 * @param:			nodes map[string]interface {}
 * @param:			rw http.ResponseWriter
 * @return:			void
 * @author:			Don Hsieh
 * @since:			09/11/2015
 * @last modified: 	09/23/2015
 * @called by:		func apiParser(rw http.ResponseWriter, req *http.Request)
 */
// func hostCreate(params map[string]interface {}, rw http.ResponseWriter) {
func hostCreate(nodes map[string]interface {}, rw http.ResponseWriter) {
	// log.Println("func hostCreate()")
	params := nodes["params"].(map[string]interface {})
	// log.Println(params)
	// log.Println(reflect.TypeOf(params))
	host := params["host"].(string)
	// log.Println(host)
	interfaces := params["interfaces"].([]interface{})
	ip := ""
	port := ""
	for i, arg := range interfaces {
		if i == 0 {
			ip = arg.(map[string]interface {})["ip"].(string)
			port = arg.(map[string]interface {})["port"].(string)
		}
	}
	groups := params["groups"].([]interface{})
	groupId := ""
	for i, group := range groups {
		if i == 0 {
			groupId = group.(map[string]interface {})["groupid"].(string)
		}
	}

	templates := params["templates"].([]interface{})
	templateId := ""
	for i, template := range templates {
		if i == 0 {
			templateId = template.(map[string]interface {})["templateid"].(string)
		}
	}

	inventory := params["inventory"].(map[string]interface {})
	macAddr := inventory["macaddress_a"].(string) + inventory["macaddress_b"].(string)

	args2 := map[string]string {
		"host": host, 
		"ip": ip,
		"port": port,
		"groupId": groupId,
		"templateId": templateId,
		"macAddr": macAddr,
	}
	log.Println(args2)
	// var args ...interface{}
	t := time.Now()
	timestamp := t.Unix()
	log.Println(timestamp)
	now := t.Format("2006-01-02 15:04:05")
	log.Println(now)
	args := []interface{}{}
	args = append(args, host)
	args = append(args, timestamp)
	args = append(args, now)
	args = append(args, now)
	log.Println(args)
	sqlcmd := "INSERT INTO graph.endpoint (endpoint,ts,t_create,t_modify) VALUES(?, ?, ?, ?)"
	// writeDb(sqlcmd, args)
	str := config.Database.Account + ":" + config.Database.Password + "@tcp(" + config.Database.Addr + ")/graph?charset=utf8"
	db, err := sql.Open("mysql", str)
	db.SetMaxOpenConns(2000)
	db.SetMaxIdleConns(1000)
	defer db.Close()

	if err != nil {
		log.Println("Oh noez, could not connect to database")
		return
	}

	stmtIns, err := db.Prepare(sqlcmd)
	if err != nil {
		log.Println(err.Error())
		return
	}
	defer stmtIns.Close() // Close the statement when we leave main() / the program terminates

	resp := nodes
	delete(resp, "params")
	var result = make(map[string]interface {})
	if sqlResult, err := stmtIns.Exec(host,timestamp,now,now); err==nil {
	// if result, err := stmtIns.Exec(args); err==nil {
		if id, err := sqlResult.LastInsertId(); err==nil {
			log.Println("insert id :", id)
			hostid := strconv.Itoa(int(id))
			// log.Println(hostid)
			// log.Println(reflect.TypeOf(hostid))
			hostids := [1]string{string(hostid)}
			result["hostids"] = hostids
		} else {
			log.Println(err.Error())
		}
	} else {
		log.Println(err.Error())
	}
	resp["result"] = result
	RenderJson(rw, resp)
}

/**
 * @function name:	func hostDelete(nodes map[string]interface {}, rw http.ResponseWriter)
 * @description:	This function gets host data for database insertion.
 * @related issues:	OWL-086, OWL-085
 * @param:			nodes map[string]interface {}
 * @param:			rw http.ResponseWriter
 * @return:			void
 * @author:			Don Hsieh
 * @since:			09/11/2015
 * @last modified: 	09/23/2015
 * @called by:		func apiParser(rw http.ResponseWriter, req *http.Request)
 */
func hostDelete(nodes map[string]interface {}, rw http.ResponseWriter) {
	params := nodes["params"].([]interface {})

	str := config.Database.Account + ":" + config.Database.Password + "@tcp(" + config.Database.Addr + ")/graph?charset=utf8"
	db, err := sql.Open("mysql", str)
	db.SetMaxOpenConns(2000)
	db.SetMaxIdleConns(1000)
	defer db.Close()

	if err != nil {
		log.Println("Oh noez, could not connect to database")
		return
	}

	sqlcmd := "DELETE FROM graph.endpoint WHERE id=?"
	stmtIns, err := db.Prepare(sqlcmd)
	if err != nil {
		log.Println(err.Error())
		return
	}
	defer stmtIns.Close() // Close the statement when we leave main() / the program terminates

	hostids := []string{}
	for _, hostId := range params {
		if result, err := stmtIns.Exec(hostId); err==nil {
			if RowsAffected, err := result.RowsAffected(); err==nil {
				if RowsAffected > 0 {
					hostids = append(hostids, hostId.(string))
				}
			} else {
				log.Println(err.Error())
			}
		} else {
			log.Println(err.Error())
		}
	}
	resp := nodes
	delete(resp, "params")
	var result = make(map[string]interface {})
	result["hostids"] = hostids
	resp["result"] = result
	RenderJson(rw, resp)
}

/**
 * @function name:	func hostUpdate(nodes map[string]interface {}, rw http.ResponseWriter)
 * @description:	This function updates host data.
 * @related issues:	OWL-086
 * @param:			nodes map[string]interface {}
 * @param:			rw http.ResponseWriter
 * @return:			void
 * @author:			Don Hsieh
 * @since:			09/23/2015
 * @last modified: 	09/24/2015
 * @called by:		func apiParser(rw http.ResponseWriter, req *http.Request)
 */
func hostUpdate(nodes map[string]interface {}, rw http.ResponseWriter) {
	log.Println("func hostUpdate()")
	params := nodes["params"].(map[string]interface {})
	hostName := params["host"].(string)
	hostId := params["hostid"].(string)
	now := time.Now().Format("2006-01-02 15:04:05")
	sqlcmd := "UPDATE graph.endpoint SET endpoint = ?, t_modify = ? WHERE id = ?"

	// writeDb(sqlcmd, args)
	str := config.Database.Account + ":" + config.Database.Password + "@tcp(" + config.Database.Addr + ")/graph?charset=utf8"
	db, err := sql.Open("mysql", str)
	db.SetMaxOpenConns(2000)
	db.SetMaxIdleConns(1000)
	defer db.Close()

	if err != nil {
		log.Println("Oh noez, could not connect to database")
		return
	}

	stmtIns, err := db.Prepare(sqlcmd)
	if err != nil {
		log.Println(err.Error())
		return
	}
	defer stmtIns.Close() // Close the statement when we leave main() / the program terminates

	var result = make(map[string]interface {})
	if sqlResult, err := stmtIns.Exec(hostName, now, hostId); err==nil {
		if RowsAffected, err := sqlResult.RowsAffected(); err==nil {
			if RowsAffected > 0 {
				hostids := [1]string{hostId}
				result["hostids"] = hostids
				log.Println("update hostId : ", hostId);
			}
		} else {
			log.Println(err.Error())
		}
	} else {
		log.Println(err.Error())
	}
	resp := nodes
	delete(resp, "params")
	resp["result"] = result
	RenderJson(rw, resp)
}

/**
 * @function name:	func hostgroupCreate(nodes map[string]interface {}, rw http.ResponseWriter)
 * @description:	This function gets hostgroup data for database insertion.
 * @related issues:	OWL-086
 * @param:			nodes map[string]interface {}
 * @param:			rw http.ResponseWriter
 * @return:			void
 * @author:			Don Hsieh
 * @since:			09/21/2015
 * @last modified: 	09/23/2015
 * @called by:		func apiParser(rw http.ResponseWriter, req *http.Request)
 */
func hostgroupCreate(nodes map[string]interface {}, rw http.ResponseWriter) {
	// log.Println("func hostgroupCreate()")
	params := nodes["params"].(map[string]interface {})
	hostgroupName := params["name"].(string)
	sqlcmd := "INSERT INTO falcon_portal.grp (grp_name) VALUES(?)"
	// writeDb(sqlcmd, args)
	str := config.Database.Account + ":" + config.Database.Password + "@tcp(" + config.Database.Addr + ")/falcon_portal?charset=utf8"
	db, err := sql.Open("mysql", str)
	db.SetMaxOpenConns(2000)
	db.SetMaxIdleConns(1000)
	defer db.Close()

	if err != nil {
		log.Println("Oh noez, could not connect to database")
		return
	}

	stmtIns, err := db.Prepare(sqlcmd)
	if err != nil {
		log.Println(err.Error())
		return
	}
	defer stmtIns.Close() // Close the statement when we leave main() / the program terminates

	var result = make(map[string]interface {})
	if sqlResult, err := stmtIns.Exec(hostgroupName); err==nil {
		if id, err := sqlResult.LastInsertId(); err==nil {
			log.Println("insert id :", id);
			groupid := strconv.Itoa(int(id))
			groupids := [1]string{string(groupid)}
			result["groupids"] = groupids
		} else {
			log.Println(err.Error())
		}
	} else {
		log.Println(err.Error())
	}
	resp := nodes
	delete(resp, "params")
	resp["result"] = result
	RenderJson(rw, resp)
}

/**
 * @function name:	func hostgroupDelete(nodes map[string]interface {}, rw http.ResponseWriter)
 * @description:	This function gets hostgroup data for database insertion.
 * @related issues:	OWL-086
 * @param:			nodes map[string]interface {}
 * @param:			rw http.ResponseWriter
 * @return:			void
 * @author:			Don Hsieh
 * @since:			09/21/2015
 * @last modified: 	09/24/2015
 * @called by:		func apiParser(rw http.ResponseWriter, req *http.Request)
 */
func hostgroupDelete(nodes map[string]interface {}, rw http.ResponseWriter) {
	// log.Println("func hostgroupDelete()")
	params := nodes["params"].([]interface {})

	str := config.Database.Account + ":" + config.Database.Password + "@tcp(" + config.Database.Addr + ")/falcon_portal?charset=utf8"
	db, err := sql.Open("mysql", str)
	db.SetMaxOpenConns(2000)
	db.SetMaxIdleConns(1000)
	defer db.Close()

	if err != nil {
		log.Println("Oh noez, could not connect to database")
		return
	}

	args := []interface{}{}
	args = append(args, "DELETE FROM falcon_portal.grp WHERE id=?")
	args = append(args, "DELETE FROM falcon_portal.grp_host WHERE grp_id=?")
	args = append(args, "DELETE FROM falcon_portal.grp_tpl WHERE grp_id=?")
	args = append(args, "DELETE FROM falcon_portal.plugin_dir WHERE grp_id=?")
	log.Println(args)
	
	groupids := []string{}
	for _, sqlcmd := range args {
		stmtIns, err := db.Prepare(sqlcmd.(string))
		if err != nil {
			log.Println(err.Error())
			return
		}
		defer stmtIns.Close() // Close the statement when we leave main() / the program terminates

		for _, hostgroupId := range params {
			if result, err := stmtIns.Exec(hostgroupId); err==nil {
				// if id, err := result.LastInsertId(); err==nil {
				if RowsAffected, err := result.RowsAffected(); err==nil {
					if RowsAffected > 0 && sqlcmd == "DELETE FROM falcon_portal.grp WHERE id=?" {
						groupids = append(groupids, hostgroupId.(string))
						log.Println("delete hostgroup id:", hostgroupId)
					}
				} else {
					log.Println(err.Error())
				}
			} else {
				log.Println(err.Error())
			}
		}
	}
	resp := nodes
	delete(resp, "params")
	var result = make(map[string]interface {})
	result["groupids"] = groupids
	resp["result"] = result
	RenderJson(rw, resp)
}

/**
 * @function name:	func hostgroupUpdate(nodes map[string]interface {}, rw http.ResponseWriter)
 * @description:	This function gets hostgroup data for database insertion.
 * @related issues:	OWL-086
 * @param:			nodes map[string]interface {}
 * @param:			rw http.ResponseWriter
 * @return:			void
 * @author:			Don Hsieh
 * @since:			09/21/2015
 * @last modified: 	09/24/2015
 * @called by:		func apiParser(rw http.ResponseWriter, req *http.Request)
 */
func hostgroupUpdate(nodes map[string]interface {}, rw http.ResponseWriter) {
	log.Println("func hostgroupUpdate()")
	params := nodes["params"].(map[string]interface {})
	hostgroupId := params["groupid"].(string)
	hostgroupName := params["name"].(string)
	sqlcmd := "UPDATE falcon_portal.grp SET grp_name = ? WHERE id = ?"

	// writeDb(sqlcmd, args)
	str := config.Database.Account + ":" + config.Database.Password + "@tcp(" + config.Database.Addr + ")/falcon_portal?charset=utf8"
	db, err := sql.Open("mysql", str)
	db.SetMaxOpenConns(2000)
	db.SetMaxIdleConns(1000)
	defer db.Close()

	if err != nil {
		log.Println("Oh noez, could not connect to database")
		return
	}

	stmtIns, err := db.Prepare(sqlcmd)
	if err != nil {
		log.Println(err.Error())
		return
	}
	defer stmtIns.Close() // Close the statement when we leave main() / the program terminates

	var result = make(map[string]interface {})
	if sqlResult, err := stmtIns.Exec(hostgroupName, hostgroupId); err==nil {
		if RowsAffected, err := sqlResult.RowsAffected(); err==nil {
			if RowsAffected > 0 {
				groupids := [1]string{hostgroupId}
				result["groupids"] = groupids
				log.Println("update groupid : ", hostgroupId);
			}
		} else {
			log.Println(err.Error())
		}
	} else {
		log.Println(err.Error())
	}
	resp := nodes
	delete(resp, "params")
	resp["result"] = result
	RenderJson(rw, resp)
}

/**
 * @function name:	func templateCreate(nodes map[string]interface {}, rw http.ResponseWriter)
 * @description:	This function gets hostgroup data for database insertion.
 * @related issues:	OWL-086
 * @param:			nodes map[string]interface {}
 * @param:			rw http.ResponseWriter
 * @return:			void
 * @author:			Don Hsieh
 * @since:			09/22/2015
 * @last modified: 	09/24/2015
 * @called by:		func apiParser(rw http.ResponseWriter, req *http.Request)
 */
func templateCreate(nodes map[string]interface {}, rw http.ResponseWriter) {
	// log.Println("func templateCreate()")
	params := nodes["params"].(map[string]interface {})
	templateName := params["host"].(string)
	user := "root"
	groups := params["groups"]
	groupid := groups.(map[string]interface{})["groupid"].(json.Number)
	hostgroupId := string(groupid)

	// writeDb(sqlcmd, args)
	str := config.Database.Account + ":" + config.Database.Password + "@tcp(" + config.Database.Addr + ")/falcon_portal?charset=utf8"
	db, err := sql.Open("mysql", str)
	db.SetMaxOpenConns(2000)
	db.SetMaxIdleConns(1000)
	defer db.Close()

	if err != nil {
		log.Println("Oh noez, could not connect to database")
		return
	}

	sqlcmd := "INSERT INTO falcon_portal.tpl (tpl_name, create_user) VALUES(?, ?)"

	stmtIns, err := db.Prepare(sqlcmd)
	if err != nil {
		log.Println(err.Error())
		return
	}
	defer stmtIns.Close() // Close the statement when we leave main() / the program terminates

	var result = make(map[string]interface {})
	if sqlResult, err := stmtIns.Exec(templateName,user); err==nil {
		if id, err := sqlResult.LastInsertId(); err==nil {
			log.Println("insert id :", id);
			templateId := strconv.Itoa(int(id))
			templateids := [1]string{string(templateId)}
			result["templateids"] = templateids

			sqlcmd = "INSERT INTO falcon_portal.grp_tpl (grp_id, tpl_id, bind_user) VALUES(?, ?, ?)"
			stmtIns, err = db.Prepare(sqlcmd)
			if err != nil {
				log.Println(err.Error())
				return
			}
			defer stmtIns.Close() // Close the statement when we leave main() / the program terminates

			if result, err := stmtIns.Exec(hostgroupId,templateId,user); err==nil {
				if id, err := result.LastInsertId(); err==nil {
					log.Println("insert id :", id);
				} else {
					log.Println(err.Error())
				}
			} else {
				log.Println(err.Error())
			}
		} else {
			log.Println(err.Error())
		}
	} else {
		log.Println(err.Error())
	}
	resp := nodes
	delete(resp, "params")
	resp["result"] = result
	RenderJson(rw, resp)
}

/**
 * @function name:	func templateDelete(nodes map[string]interface {}, rw http.ResponseWriter)
 * @description:	This function deletes template data.
 * @related issues:	OWL-086
 * @param:			nodes map[string]interface {}
 * @param:			rw http.ResponseWriter
 * @return:			void
 * @author:			Don Hsieh
 * @since:			09/22/2015
 * @last modified: 	09/24/2015
 * @called by:		func apiParser(rw http.ResponseWriter, req *http.Request)
 */
func templateDelete(nodes map[string]interface {}, rw http.ResponseWriter) {
	// log.Println("func templateDelete()")
	params := nodes["params"].([]interface {})

	str := config.Database.Account + ":" + config.Database.Password + "@tcp(" + config.Database.Addr + ")/falcon_portal?charset=utf8"
	db, err := sql.Open("mysql", str)
	db.SetMaxOpenConns(2000)
	db.SetMaxIdleConns(1000)
	defer db.Close()

	if err != nil {
		log.Println("Oh noez, could not connect to database")
		return
	}

	args := []interface{}{}
	args = append(args, "DELETE FROM falcon_portal.tpl WHERE id=?")
	args = append(args, "DELETE FROM falcon_portal.grp_tpl WHERE tpl_id=?")
	log.Println(args)
	
	templateids := []string{}
	for _, sqlcmd := range args {
		log.Println(sqlcmd)
		stmtIns, err := db.Prepare(sqlcmd.(string))
		if err != nil {
			log.Println(err.Error())
			return
		}
		defer stmtIns.Close() // Close the statement when we leave main() / the program terminates

		for _, templateId := range params {
			log.Println(templateId)
			if result, err := stmtIns.Exec(templateId); err==nil {
				if RowsAffected, err := result.RowsAffected(); err==nil {
					if RowsAffected > 0 {
						templateids = append(templateids, templateId.(string))
					}
					log.Println("delete id:", templateId);
				} else {
					log.Println(err.Error())
				}
			} else {
				log.Println(err.Error())
			}
		}
	}
	resp := nodes
	delete(resp, "params")
	var result = make(map[string]interface {})
	result["templateids"] = templateids
	resp["result"] = result
	RenderJson(rw, resp)
}

/**
 * @function name:	func templateUpdate(nodes map[string]interface {}, rw http.ResponseWriter)
 * @description:	This function gets hostgroup data for database insertion.
 * @related issues:	OWL-086
 * @param:			nodes map[string]interface {}
 * @param:			rw http.ResponseWriter
 * @return:			void
 * @author:			Don Hsieh
 * @since:			09/22/2015
 * @last modified: 	09/24/2015
 * @called by:		func apiParser(rw http.ResponseWriter, req *http.Request)
 */
func templateUpdate(nodes map[string]interface {}, rw http.ResponseWriter) {
	// log.Println("func templateUpdate()")
	params := nodes["params"].(map[string]interface {})
	templateId := params["templateid"].(string)
	templateName := params["name"].(string)
	// log.Println(templateId)
	// log.Println(templateName)
	sqlcmd := "UPDATE falcon_portal.tpl SET tpl_name = ? WHERE id = ?"

	// writeDb(sqlcmd, args)
	str := config.Database.Account + ":" + config.Database.Password + "@tcp(" + config.Database.Addr + ")/falcon_portal?charset=utf8"
	db, err := sql.Open("mysql", str)
	db.SetMaxOpenConns(2000)
	db.SetMaxIdleConns(1000)
	defer db.Close()

	if err != nil {
		log.Println("Oh noez, could not connect to database")
		return
	}

	stmtIns, err := db.Prepare(sqlcmd)
	if err != nil {
		log.Println(err.Error())
		return
	}
	defer stmtIns.Close() // Close the statement when we leave main() / the program terminates

	templateids := []string{}
	if result, err := stmtIns.Exec(templateName, templateId); err==nil {
	// if result, err := stmtIns.Exec(args); err==nil {
		if RowsAffected, err := result.RowsAffected(); err==nil {
			if RowsAffected > 0 {
				templateids = append(templateids, templateId)
				log.Println("update groupid : ", templateId);
			}
		} else {
			log.Println(err.Error())
		}
	} else {
		log.Println(err.Error())
	}
	resp := nodes
	delete(resp, "params")
	var result = make(map[string]interface {})
	result["templateids"] = templateids
	resp["result"] = result
	RenderJson(rw, resp)
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
 * @last modified: 	09/23/2015
 * @called by:		http.HandleFunc("/api", apiParser)
 *					 in func main()
 */
func apiParser(rw http.ResponseWriter, req *http.Request) {
	buf := new(bytes.Buffer)
	buf.ReadFrom(req.Body)
	s := buf.String() // Does a complete copy of the bytes in the buffer.
	log.Println("s =", s)

	json, err := simplejson.NewJson(buf.Bytes())
	if err != nil {
		log.Println(err.Error())
	}

	f, err := os.OpenFile("falcon_api.log", os.O_RDWR | os.O_CREATE | os.O_APPEND, 0666)
	if err != nil {
		log.Fatal("error opening file: %v", err)
	}
	defer f.Close()
	log.SetOutput(f)

	var nodes = make(map[string]interface {})
	nodes, _ = json.Map()

	method := nodes["method"]
	log.Println(method)
	delete(nodes, "method")
	delete(nodes, "auth")

	if method == "host.create" {
		hostCreate(nodes, rw)
	} else if method == "host.delete" {
		hostDelete(nodes, rw)
	} else if method == "host.update" {
		hostUpdate(nodes, rw)
	} else if method == "host.exists" {
		// hostExist(params)
	} else if method == "hostgroup.create" {
		hostgroupCreate(nodes, rw)
	} else if method == "hostgroup.delete" {
		hostgroupDelete(nodes, rw)
	} else if method == "hostgroup.update" {
		hostgroupUpdate(nodes, rw)
	} else if method == "hostgroup.exists" {
		// hostgroupExist(params)
	} else if method == "template.create" {
		templateCreate(nodes, rw)
	} else if method == "template.delete" {
		templateDelete(nodes, rw)
	} else if method == "template.update" {
		templateUpdate(nodes, rw)
	} else if method == "template.exists" {
		// templateExist(params)
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

