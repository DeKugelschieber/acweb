package api

import (
	"encoding/json"
	"net/http"
	"strconv"

	log "github.com/sirupsen/logrus"

	"github.com/assetto-corsa-web/acweb/config"
	"github.com/assetto-corsa-web/acweb/instance"
	"github.com/assetto-corsa-web/acweb/model"
	"github.com/assetto-corsa-web/acweb/resp"
	"github.com/assetto-corsa-web/acweb/session"
	"github.com/assetto-corsa-web/acweb/settings"
	"github.com/assetto-corsa-web/acweb/user"
)

func UserHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		AddEditUser(w, r)
	} else if r.Method == "DELETE" {
		RemoveUser(w, r)
	} else if r.Method == "GET" {
		if r.URL.Query().Get("id") == "" {
			GetAllUser(w, r)
		} else {
			GetUser(w, r)
		}
	}
}

func SettingsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		SaveSettings(w, r)
	} else if r.Method == "GET" {
		GetSettings(w, r)
	}
}

func ConfigurationHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		AddEditConfiguration(w, r)
	} else if r.Method == "DELETE" {
		RemoveConfiguration(w, r)
	} else if r.Method == "GET" {
		if r.URL.Query().Get("id") == "" {
			GetAllConfigurations(w, r)
		} else {
			dl := r.URL.Query().Get("dl")
			if dl != "" {
				downloadConfigurationHandler(w, r, dl)
			} else {
				GetConfiguration(w, r)
			}
		}
	}
}

func InstanceHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		StartInstance(w, r)
	} else if r.Method == "DELETE" {
		StopInstance(w, r)
	} else if r.Method == "GET" {
		GetAllInstances(w, r)
	}
}

func InstanceLogHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		fileName := r.URL.Query().Get("file")
		if fileName == "" {
			GetAllInstanceLogs(w, r)
		} else {
			if r.URL.Query().Get("dl") != "" {
				downloadLogHandler(w, r, fileName)
			} else {
				GetInstanceLog(w, r, fileName)
			}
		}
	} else if r.Method == "DELETE" {
		deleteLogHandler(w, r)
	}
}

func CheckSession(w http.ResponseWriter, r *http.Request) {
	s, _ := session.GetCurrentSession(r)

	if s.Active() {
		var id int64

		if err := s.Get("user_id", &id); err != nil {
			log.WithFields(log.Fields{"err": err}).Error("Error reading user ID")
			resp.Error(w, 1, "Error reading user ID", nil)
			return
		}

		resp.Success(w, 0, "", struct {
			Id int64 `json:"user_id"`
		}{id})
	} else {
		// don't log this
		resp.Log = false
		resp.Failure(w, 3, "Not logged in", nil)
		resp.Log = true
	}
}

func Login(w http.ResponseWriter, r *http.Request) {
	req := struct {
		Login string `json:"login"`
		Pwd   string `json:"pwd"`
	}{}

	if decode(w, r, &req) {
		return
	}

	user, err := user.Login(req.Login, req.Pwd)

	if iserror(w, err) {
		return
	}

	// start session
	s, err := session.NewSession(w, r)

	if err != nil {
		log.WithFields(log.Fields{"err": err}).Error("Error starting session on login")
		resp.Error(w, 3, err.Error(), nil)
		return
	}

	s.Set("user_id", user.Id)
	s.Set("admin", user.Admin)
	s.Set("moderator", user.Moderator)

	if err := s.Save(); err != nil {
		log.WithFields(log.Fields{"err": err}).Error("Error saving session on login")
		resp.Error(w, 4, err.Error(), nil)
		return
	}

	resp := struct {
		UserId int64 `json:"user_id"`
	}{user.Id}
	respJson, _ := json.Marshal(resp)
	w.Write(respJson)
}

func Logout(w http.ResponseWriter, r *http.Request) {
	s, err := session.GetCurrentSession(r)

	if !s.Active() {
		log.WithFields(log.Fields{"err": err}).Error("Session not found on logout")
		resp.Error(w, 1, "Session not found", err)
		return
	}

	if err := s.Destroy(w, r); err != nil {
		log.WithFields(log.Fields{"err": err}).Error("Error destroying user session on logout")
		resp.Error(w, 2, "Error destroying user session", nil)
		return
	}

	success(w)
}

func AddEditUser(w http.ResponseWriter, r *http.Request) {
	if !isadmin(r) {
		denyAccess(w)
		return
	}

	req := struct {
		Id        int64  `json:"id"`
		Login     string `json:"login"`
		Email     string `json:"email"`
		Pwd1      string `json:"pwd1"`
		Pwd2      string `json:"pwd2"`
		Admin     bool   `json:"admin"`
		Moderator bool   `json:"moderator"`
	}{}

	if decode(w, r, &req) {
		return
	}

	err := user.AddEditUser(req.Id,
		req.Login,
		req.Email,
		req.Pwd1,
		req.Pwd2,
		req.Admin,
		req.Moderator)

	if iserror(w, err) {
		return
	}

	success(w)
}

func RemoveUser(w http.ResponseWriter, r *http.Request) {
	if !isadmin(r) {
		denyAccess(w)
		return
	}

	id, err := strconv.Atoi(r.URL.Query().Get("id"))

	if err != nil {
		resp.Error(w, 100, err.Error(), nil)
		return
	}

	err = user.RemoveUser(int64(id))

	if iserror(w, err) {
		return
	}

	success(w)
}

func GetAllUser(w http.ResponseWriter, r *http.Request) {
	user, err := user.GetAllUser()

	if iserror(w, err) {
		return
	}

	if !isadmin(r) {
		for i := range user {
			user[i].Email = ""
		}
	}

	resp, _ := json.Marshal(user)
	w.Write(resp)
}

func GetUser(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(r.URL.Query().Get("id"))

	if err != nil {
		resp.Error(w, 100, err.Error(), nil)
		return
	}

	user, err := user.GetUser(int64(id))

	if iserror(w, err) {
		return
	}

	if !isadmin(r) {
		user.Email = ""
	}

	resp, _ := json.Marshal(user)
	w.Write(resp)
}

func SaveSettings(w http.ResponseWriter, r *http.Request) {
	if !isadmin(r) {
		denyAccess(w)
		return
	}

	req := struct {
		Folder     string `json:"folder"`
		Executable string `json:"executable"`
		Args       string `json:"args"`
	}{}

	if decode(w, r, &req) {
		return
	}

	err := settings.SaveSettings(req.Folder, req.Executable, req.Args)

	if iserror(w, err) {
		return
	}

	success(w)
}

func GetSettings(w http.ResponseWriter, r *http.Request) {
	settings := settings.GetSettings()
	resp, _ := json.Marshal(settings)
	w.Write(resp)
}

func AddEditConfiguration(w http.ResponseWriter, r *http.Request) {
	if !ismoderator(r) {
		denyAccess(w)
		return
	}

	var req model.Configuration

	if decode(w, r, &req) {
		return
	}

	err := config.AddEditConfiguration(&req)

	if iserror(w, err) {
		return
	}

	success(w)
}

func RemoveConfiguration(w http.ResponseWriter, r *http.Request) {
	if !ismoderator(r) {
		denyAccess(w)
		return
	}

	id, err := strconv.Atoi(r.URL.Query().Get("id"))

	if err != nil {
		resp.Error(w, 100, err.Error(), nil)
		return
	}

	err = config.RemoveConfiguration(int64(id))

	if iserror(w, err) {
		return
	}

	success(w)
}

func GetAllConfigurations(w http.ResponseWriter, r *http.Request) {
	configs, err := config.GetAllConfigurations()

	if iserror(w, err) {
		return
	}

	resp, _ := json.Marshal(configs)
	w.Write(resp)
}

func GetConfiguration(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(r.URL.Query().Get("id"))

	if err != nil {
		resp.Error(w, 100, err.Error(), nil)
		return
	}

	config, err := config.GetConfiguration(int64(id))
	if iserror(w, err) {
		return
	}

	resp, _ := json.Marshal(config)
	w.Write(resp)
}

func downloadConfigurationHandler(w http.ResponseWriter, r *http.Request, dlType string) {
	id, err := strconv.Atoi(r.URL.Query().Get("id"))
	if err != nil {
		resp.Error(w, 100, err.Error(), nil)
		return
	}

	config, err := config.GetConfiguration(int64(id))
	if iserror(w, err) {
		return
	}

	w.Header().Set("Content-Disposition", "attachment; filename=\""+config.Name+".zip\"")
	w.Header().Set("Content-Type", "application/zip")

	if dlType == "1" {
		err = instance.ZipConfiguration(config, w)
	} else if dlType == "2" {
		err = instance.ZipInstanceFiles(config, w)
	} else {
		resp.Error(w, 100, "Invalid download option", nil)
		return
	}

	if iserror(w, err) {
		return
	}

	success(w)
}

func downloadLogHandler(w http.ResponseWriter, r *http.Request, fileName string) {
	w.Header().Set("Content-Disposition", "attachment; filename=\""+fileName+".zip\"")
	w.Header().Set("Content-Type", "application/zip")

	err := instance.ZipLogFile(fileName, w)
	if iserror(w, err) {
		return
	}

	success(w)
}

func deleteLogHandler(w http.ResponseWriter, r *http.Request) {
	filename := r.URL.Query().Get("filename")

	if filename != "" {
		if err := instance.DeleteLogFile(filename); iserror(w, err) {
			return
		}
	} else {
		if err := instance.DeleteAllLogFiles(); iserror(w, err) {
			return
		}
	}

	success(w)
}

func GetAvailableTracks(w http.ResponseWriter, r *http.Request) {
	tracks, err := config.GetAvailableTracks()

	if iserror(w, err) {
		return
	}

	resp, _ := json.Marshal(tracks)
	w.Write(resp)
}

func GetAvailableCars(w http.ResponseWriter, r *http.Request) {
	cars, err := config.GetAvailableCars()

	if iserror(w, err) {
		return
	}

	resp, _ := json.Marshal(cars)
	w.Write(resp)
}

func StartInstance(w http.ResponseWriter, r *http.Request) {
	if !ismoderator(r) {
		denyAccess(w)
		return
	}

	req := struct {
		Name          string `json:"name"`
		Configuration int64  `json:"config"`
		ScriptBefore  string `json:"script_before"`
		ScriptAfter   string `json:"script_after"`
	}{}

	if decode(w, r, &req) {
		return
	}

	err := instance.StartInstance(req.Name, req.Configuration, req.ScriptBefore, req.ScriptAfter)

	if iserror(w, err) {
		return
	}

	success(w)
}

func StopInstance(w http.ResponseWriter, r *http.Request) {
	if !ismoderator(r) {
		denyAccess(w)
		return
	}

	pid, err := strconv.Atoi(r.URL.Query().Get("pid"))

	if err != nil {
		resp.Error(w, 100, err.Error(), nil)
		return
	}

	err = instance.StopInstance(pid)

	if iserror(w, err) {
		return
	}

	success(w)
}

func GetAllInstances(w http.ResponseWriter, r *http.Request) {
	instances := instance.GetAllInstances()
	resp, _ := json.Marshal(instances)
	w.Write(resp)
}

func GetAllInstanceLogs(w http.ResponseWriter, r *http.Request) {
	logs, err := instance.GetAllInstanceLogs()
	if iserror(w, err) {
		return
	}

	resp, _ := json.Marshal(logs)
	w.Write(resp)
}

func GetInstanceLog(w http.ResponseWriter, r *http.Request, file string) {
	log, err := instance.GetInstanceLog(file)
	if iserror(w, err) {
		return
	}

	resp, _ := json.Marshal(log)
	w.Write(resp)
}
