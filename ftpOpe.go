package main

import (
	"github.com/jlaffaye/ftp"
	"path/filepath"
	"os"
	"io/ioutil"
	"fmt"
	"strings"
	"bytes"
	"time"
	json "github.com/bitly/go-simplejson"
	log "github.com/alecthomas/log4go"
	)

///////////////////////////////////// FtpBase 
type FtpBase struct {
	ip, usr, pwd string
	homePath string
	conn *ftp.ServerConn
}

func (f* FtpBase) Conn() (err error) {
	conn, err := ftp.Connect(f.ip)
	if err != nil {
		log.Info("ftp.Connect[%s] %s", f.ip, err)
		return err
	}
	
	err = conn.Login(f.usr, f.pwd)
	if err != nil {
		log.Info("ftp.Login[%s:%s] %s", f.usr, f.pwd, err)
		return err
	}
	
	f.conn = conn
	f.homePath, err = conn.CurrentDir()
	if err != nil {
		log.Info("conn.CurrentDir(): [%s][%s:%s] %s", f.ip, f.usr, f.pwd, err)
		return err
	}
	
	return nil
}

func (f* FtpBase) ReConn() (err error) {
	f.Close()
	for {
		err = f.Conn()
		if err == nil {
			return nil
		}
		time.Sleep(time.Second*10)
		log4.Warn("Reconn err[%s], sleep 10s", err)
	}	
}

func (f* FtpBase) Close() (err error) {
	if f.conn != nil {
		f.conn.Logout()
		f.conn.Quit()
		f.conn = nil
	}
	
	return nil
}

func (f* FtpBase) Noop() {
	for {
		if f.conn == nil {
			f.ReConn()
		}
		
		err := f.conn.NoOp()
		if err == nil {
			break
		} else {
			log4.Error("Noop err[%s]", err)
			f.ReConn()
		}
	}
}

func (f FtpBase) ListLocal(folder string, m map[string]string) (err error) {
	err = filepath.Walk(folder, func(path string, f os.FileInfo, err error) error {
        if ( f == nil ) {
			return err
		} else if f.IsDir() {
			m[path] = "folder"
		} else {
			if f.Mode() & os.ModeSymlink  == 0 { // 排除link文件
				m[path] = "file"
			}
		}
		return nil
	})
	
	return 
}

func (f FtpBase) ListServer(folder string, m map[string]string) (err error) {
	if folder[len(folder)-1] == '/' {
		folder = folder[:len(folder)-1]
	}
		
	// 文件
	err = f.conn.ChangeDir(folder)
	if  err != nil { //failed
		i:= len(folder)-1
		for ; folder[i] != '/'; i-- {
		}
		parentDir := folder[:i+1]
		childName := folder[i+1:]
		entrys2, _ := f.conn.List(parentDir)
		for k := range entrys2 {
			if entrys2[k].Name == childName {
				m[folder] = "file"
				return nil
			}
		}
		
		return nil
	}
	
	// 目录
	err = f.conn.ChangeDir(f.homePath)
	if err != nil {
		log.Info("f.conn.ChangeDir[%s] error %s", f.homePath, err)
		return
	}
	entrys, err := f.conn.List(folder)
	if len(entrys) >= 1 {
		m[folder] = "folder"
		for i := range(entrys) {
			switch entrys[i].Type {
				case ftp.EntryTypeFile: {
					m[folder+"/"+entrys[i].Name] = "file"
				}
				case ftp.EntryTypeFolder: {
					m[folder+"/"+entrys[i].Name] = "folder"
					f.ListServer(folder+"/"+entrys[i].Name, m)
				}
			}
		}
	} else {
		i:= len(folder)-1
		for ; folder[i] != '/'; i-- {
		}
		parentDir := folder[:i+1]
		entrys2, _ := f.conn.List(parentDir)
		for i := range entrys2 {
			if folder == parentDir +entrys2[i].Name {
				m[folder] = "folder"
				break;
			}
		}
	}
	
	return
}

func (f FtpBase) CreateFolderLocal(folder string) (err error) {
	_, err = os.Stat(folder) 
	if err == nil || os.IsExist(err) {
		return nil
	} else {
		err = os.MkdirAll(folder, 0766)
		if err != nil {
			log.Info("CreateFolderLocal[%s] error[%s]", folder, err)
		}
		return err
	}
} 

func (f FtpBase) CreateFolderServer(folder string) (err error) {
//	fmt.Printf("CreateFolderServer: [%s] \n", folder)
	a := strings.SplitN(folder, "/", -1)
	var path_tmp string
	var p string
	var i int
	
	for i = range a {
		path_tmp += a[i] + "/"
		err = f.conn.MakeDir(path_tmp)
		if i < len(a) -1 {
			p += a[i] + "/"
		}
	}
	
	entrys, err := f.conn.List(p)
	if err != nil {
		log.Info("f.conn.List: [%s] error[%s]", p, err)
		return err
	}
	
	for j := range entrys {
		if entrys[j].Name == a[i] {
			return nil
		} 
	}

	return nil
}

func (f FtpBase) GetFile(srcFile string, dstFile string) (err error) {
//	fmt.Printf("GetFile: srcFile[%s] dstFile[%s] \n", srcFile, dstFile)
	// create folder
	i := len(dstFile)-1
	for ; i >= 0 && dstFile[i] != '/'; i-- {
	}
	f.CreateFolderLocal(dstFile[:i])
	
	// create file
	r, err := f.conn.Retr(srcFile)
	if err != nil {
		log.Info("GetFile: f.conn.Retr[%s] error[%s]", srcFile, err)
		return err
	} 

	defer r.Close()
	buf, err := ioutil.ReadAll(r)
	if err == nil {
		err = ioutil.WriteFile(dstFile, buf, 0766) //os.ModeAppend
		if err != nil {
			log.Info("ioutil.WriteFile: dstFile[%s] err[%s]", dstFile, err)
		}
	}
	
	return err
}

// 判断本地文件是否存在
func (f FtpBase) isFileExistLocal(name string) bool {
	_, err := os.Stat(name)
	return err == nil || os.IsExist(err)
}

func (f FtpBase) PutFile(srcFile string, dstFile string) (err error) {
//	fmt.Printf("PutFile: srcFile[%s] dstFile[%s] \n", srcFile, dstFile)
	if f.isFileExistLocal(srcFile) == false {
		log.Info("isFileExistLocal: srcFile[%s] not exist at local", srcFile)
		return os.ErrNotExist
	}
		
	// 在服务器上创建目录
	if dstFile[len(dstFile)-1] == '/' {
		dstFile = dstFile[:len(dstFile)-1]
	}
	i := len(dstFile) -1
	for ; dstFile[i] != '/'; i-- {
	} 
	dstPath := dstFile[:i]
	
	err = f.CreateFolderServer(dstPath)
	if err != nil {
		log.Info("CreateFolderServer[%s] failed[err]", dstPath, err)
		return err
	}
	
	b, err := ioutil.ReadFile(srcFile)
	if err != nil {
		log.Info("read srcFile[%s] failed[%s]", srcFile, err)
		return err
	}
	// 在服务器端创建文件
	err = f.conn.Stor(dstFile, bytes.NewBufferString(string(b)))
	if err != nil {
		log.Info("stor srcName[%s] dstName[%s] failed[%s]", srcFile, dstFile, err)
	}
	return err
}

/////////////////////////////////////// FtpOpe
type FtpOpe struct {
	SrcIP    string
	SrcUsr   string
	SrcPwd   string
	DstIP    string
	DstUsr   string
	DstPwd   string
}

func (f *FtpOpe) FtpCopy(src_IP string, src_user string, src_pwd string, src_path string,
		dst_IP string, dst_user string, dst_pwd string, dst_path string) (err error){	
	tmp_dir := fmt.Sprintf("%s/%d/", "/tmp/ftp/dzhyun", time.Now().Unix())
	
	os.RemoveAll(tmp_dir)
	defer os.RemoveAll(tmp_dir)
	
	f1 := FtpBase{src_IP, src_user, src_pwd, "", nil}
	err = f1.Conn()
	if(err != nil) {
		return
	} 
	defer f1.Close()
	
	if src_path[0] != '/' {
		var dir string
		dir, err = f1.conn.CurrentDir()
		if err != nil {
			return
		}
		src_path = dir + "/" + src_path
	}
	m1 := make(map[string]string)
	err = f1.ListServer(src_path, m1)
	if err != nil {
		return 
	}
	
	for k, v := range m1 {
		switch v {			
			case "file": {
				if k == src_path {
					tmp_len := len(k)-1
					for ; tmp_len > 0 && k[tmp_len] != '/'; tmp_len--{
						
					}
					f1.GetFile(k, tmp_dir+k[tmp_len:])
				} else {
					f1.GetFile(k, tmp_dir+k[len(src_path):])
				}
			}
			case "folder": {
				f1.CreateFolderLocal(tmp_dir+k[len(src_path):])
			}
		}
		
	}
	
	fmt.Println("---------------")
	if dst_path[len(dst_path)-1] == '/' {
		dst_path = dst_path[:len(dst_path)-1]
	}
	
	f2 := FtpBase{dst_IP, dst_user, dst_pwd, "", nil}
	err = f2.Conn()
	if(err != nil) {
		return
	} 
	defer f2.Close()
	
	if dst_path[0] != '/' {
		var dir string
		dir, err = f2.conn.CurrentDir()
		if err != nil {
			return
		}
		dst_path = dir + "/" + dst_path
	}

	m2 := make(map[string]string)
	f2.ListLocal(tmp_dir, m2)
	for k, v := range m2 {
		switch v {
			case "file": f2.PutFile(k, dst_path+"/"+k[len(tmp_dir):])
			case "folder":f2.CreateFolderServer(dst_path+"/"+k[len(tmp_dir):])
		}
	}
	
	return nil
}

func (f FtpOpe) FtpWalkDir(IP string, user string, pwd string, path string) (js *json.Json, err error){
	lf := &FtpBase{IP, user, pwd, "", nil}
	err = lf.Conn()
	if err != nil {
		return
	}
	
	if path[0] != '/' {
		var dir string
		dir, err = lf.conn.CurrentDir()
		if err != nil {
			return
		}
		path = dir + "/" + path 
	}
	
	js, err = json.NewJson([]byte(`{}`))
	entrys, err := lf.conn.List(path) 
	for i := range entrys {
		switch entrys[i].Type {
			case ftp.EntryTypeFile: js.Set(entrys[i].Name, "file")
			case ftp.EntryTypeFolder: js.Set(entrys[i].Name, "folder")
		}
	}
	
	lf.Close()
	return 
}

//////////////////////// test

func Test_WalkDir() {
	// (1) 绝对路径
	f := FtpOpe{}
	
	js, _ := f.FtpWalkDir("10.15.107.74:21", "dzhyunftp", "123456", "/home/dzhyunftp")		
	b, _ := js.MarshalJSON()
	fmt.Println(string(b))
	// (2) 相对路径
	js, _ = f.FtpWalkDir("10.15.107.74:21", "dzhyunftp", "123456", "push2")		
	b, _ = js.MarshalJSON()
	fmt.Println(string(b))
}

func Test_FtpCopy() {
	// (1)相对路径
	f := FtpOpe{}
	f.FtpCopy("10.15.107.74:21", "dzhyunftp", "123456", "/home/dzhyunftp/push1", 
	"10.15.107.75:21", "yunftp75", "123456", "/home/yunftp75/push1")
	fmt.Printf("end1 \n")
	
	// (2)绝对路径 
	f.FtpCopy("10.15.107.74:21", "dzhyunftp", "123456", "push2", 
	"10.15.107.75:21", "yunftp75", "123456", "push2")
	fmt.Printf("end2 \n")
	time.Sleep(time.Millisecond*3000)
}

func main() {
	Test_WalkDir()
	Test_FtpCopy()
}
