package utils

import (
	"github.com/ftp"
	"fmt"
	"io/ioutil"
	"time"
	"os"
	"bytes"
	"strings"
	"path/filepath"
)

// 列举服务器目录下所有文件
func listAllFilesServer(conn *ftp.ServerConn, path string) (entrys []*ftp.Entry, err error){
	return conn.List(path)
}

// 判断本地文件是否存在
func isFileExistLocal(name string) bool {
	_, err := os.Stat(name)
	return err == nil || os.IsExist(err)
}

// 在本地创建目录 如"/home/ftp/abc/def"
func createPathLocal(path string) error {
	err := os.MkdirAll(path, 0766)
	if err != nil {
		fmt.Printf("createPathLocal[%s] error[%s] \n", path, err)
	}
	return err
}

// 在服务器端创建目录，必须一级一级的创建
func createFolderServer(conn *ftp.ServerConn, path string) (err error) {
	a := strings.SplitN(path, "/", -1)
	var path_tmp string
	var p string
	var i int
	
	for i = range a {
		path_tmp += a[i] + "/"
		err = conn.MakeDir(path_tmp)
		if i < len(a) -1 {
			p += a[i] + "/"
		}
	}
	
	entrys, err := conn.List(p)
	if err != nil {
		fmt.Printf("conn.List[%s] error[%s] \n", p, err)
		return err
	}
	
	for j := range entrys {
		if entrys[j].Name == a[i] {
//			fmt.Printf("mkdir [%s] success \n",path)
			return nil
		} 
	}

	return nil
}

// 在本地创建文件
func createFileLocal(name string) error {
	i := len(name) -1
	for ; name[i] != '/'; i-- {
	} 
	path := name[:i]
	
	err := createFolderLocal(path)
	if err != nil {
		fmt.Printf("createFolderLocal[%s] failed[err] \n", path, err)
		return err
	}

	if isFileExistLocal(name) {
		err := os.Rename(name, name + "." + fmt.Sprintf("%d", time.Now().Nanosecond())) //将旧文件备份
		if err != nil {
			fmt.Printf("Rename old file[%s] failed[err] \n", name, err)
			return err
		}
	}
	
	fs, err := os.Create(name)
	defer fs.Close()
	
	if err != nil {
		fmt.Printf("create file[%s] failed[err] \n", name, err)
		return err
	}

	return nil
}

// 从服务器上取文件
func getFileFromServer(conn *ftp.ServerConn, srcFileName string, dstFileName string) error{
//	fmt.Printf("getFileFromServer: srcFileName[%s], dstFileName[%s] \n", srcFileName, dstFileName)
	r, err := conn.Retr(srcFileName)
	if err != nil {
		fmt.Printf("getFileFromServer: conn.Retr[%s] error[%s] \n", srcFileName, err)
		return err
	} 

	defer r.Close()
	buf, err := ioutil.ReadAll(r)
	if err != nil {
		return err
	} else {
		createFileLocal(dstFileName)
		ioutil.WriteFile(dstFileName, buf, os.ModeAppend)
	}
	
	return nil
}

// 将文件放到服务器上
func putFileToServer(conn *ftp.ServerConn, srcFileName string, dstFileName string) error {
//	fmt.Printf("putFileToServer: srcFileName[%s], dstFileName[%s] \n", srcFileName, dstFileName)
	if isFileExistLocal(srcFileName) == false {
		fmt.Printf("putFileToServer: srcFileName[%s] not exist at local\n", srcFileName)
		return os.ErrNotExist
	}
	
	b, err := ioutil.ReadFile(srcFileName)
	if err != nil {
		fmt.Printf("read srcFileName[%s] failed[%s] \n", srcFileName, err)
		return err
	}
	
	// 在服务器上创建目录
	i := len(dstFileName) -1
	for ; dstFileName[i] != '/'; i-- {
	} 
	dstPath := dstFileName[:i]
	
	err = createFolderServer(conn, dstPath)
	if err != nil {
		fmt.Printf("createPathLocal[%s] failed[err] \n", dstPath, err)
		return err
	}
	
	// 在服务器端创建文件
	err = conn.Stor(dstFileName, bytes.NewBufferString(string(b)))
	if err != nil {
		fmt.Printf("stor srcName[%s] dstName[%s] failed[%s] \n", srcFileName, dstFileName, err)
	}
	return err
}

// 在本地创建folder
func getFolderFromServer(conn *ftp.ServerConn, srcFolder string, dstFolder string) error{
//	fmt.Printf("getFolderFromServer: srcFolder[%s], dstFolder[%s] \n", srcFolder, dstFolder)
	createPathLocal(dstFolder)
	
	return nil
}

// 遍历本地目录，将子目录下的文件，放到m中
func walkFolderLocal(folder string, m map[string]string) (err error) {
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

// 将本地目录拷贝到服务器
func putFolderToServer(conn *ftp.ServerConn, srcFolder string, dstFolder string) error {
//	fmt.Printf("putFolderToServer: srcFolder[%s], dstFolder[%s] \n", srcFolder, dstFolder)
	if dstFolder[len(dstFolder)-1] == '/' {
		dstFolder = dstFolder[:len(dstFolder)-1]
	}
	
	if srcFolder[len(srcFolder)-1] == '/' {
		srcFolder = srcFolder[:len(srcFolder)-1]
	}
	
	m := make(map[string]string)
	err := walkFolderLocal(srcFolder, m)
	delete(m, srcFolder) //不拷贝父路径
        
//    for i := range m {
//    	fmt.Printf("putFolderToServer: [%s] [%s] \n", m[i], i)
//    }
    
    for i := range m {
    	tmpPath := i[len(srcFolder)+1:]
		switch m[i] {
			case "folder": createFolderServer(conn, dstFolder+"/"+tmpPath)
			case "file": putFileToServer(conn, i, dstFolder+"/"+tmpPath)
		}
    }
	
	return err
}

// 在本地创建文件夹
func createFolderLocal(folder string) error {
//	fmt.Printf("createFolderLocal:[%s]\n", folder)
	_, err := os.Stat(folder) 
	if err == nil || os.IsExist(err) {
		return nil
	} else {
		defer createPathLocal(folder)
	}
	
	for i := len(folder)-1; i > 0; i-- {
		if folder[i] == '/' {
			_, err = os.Stat(folder[0:i])
			if err == nil || os.IsExist(err) {
				return nil
			} else {
				defer createPathLocal(folder[0:i])
			}
		}
	}

	return nil
}

//获取服务器上所有的文件和目录
func getAllEntrysServer(conn *ftp.ServerConn, src string, m map[string]string) (err error) {
	i:= len(src)-1
	for ; src[i] != '/'; i-- {
	}
	parentDir := src[:i+1]
	
	entrys, err := conn.List(src)
	for i := range entrys {
		switch entrys[i].Type {
			case ftp.EntryTypeFile: {
				if parentDir+entrys[i].Name == src {
					m[src] = "file"
				} else {
					m[src+"/"+entrys[i].Name] = "file"
				}
			}
			case ftp.EntryTypeFolder: {
				m[src+"/"+entrys[i].Name] = "folder"
				getAllEntrysServer(conn, src+"/"+entrys[i].Name, m)
			}
		}
	}
	
	return
}

func getFromServer(conn *ftp.ServerConn, src string, dst string) (err error) {
	if dst[len(dst)-1] == '/' {
		dst = dst[:len(dst)-1]
	}
	
	if src[len(src)-1] == '/' {
		src = src[:len(src)-1]
	}
		
	m := make(map[string]string)
	err = getAllEntrysServer(conn, src, m)

	if m[src] != "file" {
		delete(m, src)
	}

	for i := range m {
		switch m[i] {
			case "file": {
				if i == src {
					tmp_len := len(src)-1
					for ; tmp_len > 0 && src[tmp_len] != '/'; tmp_len--{
						
					}
					getFileFromServer(conn, src, dst+i[tmp_len:])
				} else {
					getFileFromServer(conn, i, dst + i[len(src):]) //文件
				}
			}
			case "folder": {
				createFolderLocal(dst+i[len(src):])
			}
		}
	}
	
	return 
}

func putToServer(conn *ftp.ServerConn, src string, dst string) (err error) {
//	fmt.Printf("putToServer: src[%s] dst[%s] \n", src, dst)
	state, err := os.Stat(src) 
	if err != nil {
		fmt.Printf("os.Stat[%s] error[%s] \n", src, err)
		return err
	}
	if state.IsDir() == false { //文件
		i := len(src)-1
		for ; src[i] != '/'; i-- {
			
		}
		return putFileToServer(conn, src, dst+src[i:])
	} else { //路径
		return putFolderToServer(conn, src, dst)
	}
}

type ftpOpe struct {}

// 将目录或文件，从一个ftpserver拷贝到另外一个ftpserver
func (f *ftpOpe) Ftp_copy(src_IP string, src_user string, src_pwd string, src_path string,
		dst_IP string, dst_user string, dst_pwd string, dst_path string) error{
	defer os.RemoveAll("/tmp/ftp/")
	
	conn, err := ftp.Connect(src_IP)
	if err != nil {
		fmt.Printf("ftp.Connect[%s] [%s] \n", src_IP, err)
		return err
	}
	
	err = conn.Login(src_user, src_pwd)
	if err != nil {
		fmt.Printf("conn.Login[%s] [%s] \n", src_IP, err)
		return err
	}
	
	getFromServer(conn, src_path, "/tmp/ftp/")
	conn.Logout()
	conn.Quit()

//	fmt.Println("-------------------------------------------")
	conn, err = ftp.Connect(dst_IP)
	if err != nil {
		fmt.Printf("ftp.Connect[%s] [%s] \n", dst_IP, err)
		return err
	}
	
	err = conn.Login(dst_user, dst_pwd)
	if err != nil {
		fmt.Printf("conn.Login[%s] [%s] \n", dst_IP, err)
		return err
	}

	putToServer(conn, "/tmp/ftp/", dst_path)
	conn.Logout()
	conn.Quit()
	return nil
}

// 获取目录的结构
func (f *ftpOpe) WalkDir(IP string, user string, pwd string, path string) (entrys []*ftp.Entry, error error){
	conn, err := ftp.Connect(IP)
	if err != nil {
		fmt.Printf("ftp.Connect[%s] [%s] \n", IP, err)
		return 
	}
	
	err = conn.Login(user, pwd)
	if err != nil {
		fmt.Printf("conn.Login[%s] [%s] \n", IP, err)
		return 
	}
	
	entrys, err = conn.List(path)
	conn.Logout()
	conn.Quit()
	
	return 
}
