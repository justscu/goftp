package main
//package utils

import (
	"github.com/ftp"
	"fmt"
	"io/ioutil"
	"time"
	"os"
	"bytes"
	"strings"
)

// 列举服务器目录下所有文件
func listAllFilesServer(conn *ftp.ServerConn, path string) (entrys []*ftp.Entry, err error){
	entrys, err = conn.List(path)
	if err != nil {
		return 
	}
	
	for i := range entrys {
		var fileType string
		if entrys[i].Type == ftp.EntryTypeFile {
			fileType = "file"
		} else if entrys[i].Type == ftp.EntryTypeFolder {
			fileType = "folder"
		} else if entrys[i].Type == ftp.EntryTypeLink {
			fileType = "link"
		}
		
		fileSize := fmt.Sprintf("%d", entrys[i].Size)
		filename := path + "/" + entrys[i].Name
		fmt.Printf("[%s] [%s] [%s] [%s] \n", filename, fileType, fileSize, entrys[i].Time)
	
		// 递归枚举所有的子目录	
//		if entrys[i].Type == ftp.EntryTypeFolder {
//			ListAllFilesServer(conn, filename)
//		}
	}
	
	return
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


// 在服务器端创建目录
func createPathServer(conn *ftp.ServerConn, path string) (err error) {
	a := strings.SplitN(path, "/", -1)
	var path_tmp string
	var p string
	var i int
	for i = range a {
		path_tmp += a[i] + "/"
		conn.MakeDir(path_tmp)
		
		if i < len(a) -1 {
			p += a[i] + "/"
		}
	}
	
	fmt.Println(p)
	entrys, err := conn.List(p)
	if err != nil {
		fmt.Printf("conn.List[%s] error[%s] \n", p, err)
		return err
	}
	
	for j := range entrys {
		if entrys[j].Name == a[i] {
			fmt.Printf("create [%s] success \n",path)
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
	
	err := createPathLocal(path)
	if err != nil {
		fmt.Printf("createPathLocal[%s] failed[err] \n", path, err)
		return err
	}

	if isFileExistLocal(name) {
		err := os.Rename(name, name + "." + fmt.Sprintf("%d", time.Now().Nanosecond()))
		if err != nil {
			fmt.Printf("Rename old file[%s] failed[err] \n", name, err)
			return err
		}
	}
	
	fs, err := os.Create(name)
	if err != nil {
		fmt.Printf("create file[%s] failed[err] \n", name, err)
		return err
	}
	
	fs.Close()
	return nil
}

// 从服务器上取文件
func getFileFromServer(conn *ftp.ServerConn, srcName string, dstName string) error{
	r, err := conn.Retr(srcName)
	if err != nil {
		return err
	} 

	defer r.Close()
	buf, err := ioutil.ReadAll(r)
	if err != nil {
		return err
	} else {
		createFileLocal(dstName)
		ioutil.WriteFile(dstName, buf, os.ModeAppend)
//			fmt.Printf("\n%s\n", string(buf))
	}
	
	return nil
}

// 将文件放到服务器上
func putFileToServer(conn *ftp.ServerConn, srcName string, dstName string) error {
	if isFileExistLocal(srcName) == false {
		fmt.Printf("srcName[%s] not exist at local\n", srcName)
		return os.ErrNotExist
	}
	
	b, err := ioutil.ReadFile(srcName)
	if err != nil {
		fmt.Printf("read srcName[%s] failed[%s] \n", srcName, err)
		return err
	}
	
	// 在服务器上创建目录
	i := len(dstName) -1
	for ; dstName[i] != '/'; i-- {
	} 
	dstPath := dstName[:i]
	
	err = createPathServer(conn, dstPath)
	if err != nil {
		fmt.Printf("createPathLocal[%s] failed[err] \n", dstPath, err)
		return err
	}
	
	// 在服务器端创建文件
	err = conn.Stor(dstName, bytes.NewBufferString(string(b)))
	if err != nil {
		fmt.Printf("stor srcName[%s] dstName[%s] failed[%s] \n", srcName, dstName, err)
	}
	return err
}

// 注意：测试该函数，需要修改对应的: ip/port, 用户名/密码, 目录名称
// func TestFtpTools() {
func main() {
	fmt.Printf("main \n")
	conn, err := ftp.Connect("10.15.107.74:21")
	if err != nil {
		fmt.Printf("ftp.Connect [%s] \n", err)
		return
	}
	
	err = conn.Login("dzhyunftp", "123456")
	if err != nil {
		fmt.Printf("conn.Login [%s] \n", err)
		return
	}
	
	dir, err := conn.CurrentDir()
	fmt.Printf("current dir [%s] \n", dir)
	
	//列举目录下所有文件
	listAllFilesServer(conn, dir)
	
	// 从服务器上获取文件
	srcFile := "/home/dzhyunftp/t1/t2/t3/t4/zoo_api.cpp"
	dstFile := "/home/ll/t1/t2/t3/t4/zoo_api.cpp"
	err = getFileFromServer(conn, srcFile, dstFile)
	fmt.Printf("GetFileFromServer srcFile[%s] [%s] \n", srcFile, err)
	
	srcFile = "/home/ll/pushproxy.info.2014-09-16.log"
	dstFile = "/home/dzhyunftp/t1/t2/t3/main.log"
	// 将文件放到服务器上
	err = putFileToServer(conn, srcFile, dstFile)
	fmt.Printf("putFileToServer srcFile[%s] [%s] \n", srcFile, err)
	
	fmt.Printf("end \n")
	
	time.Sleep(time.Millisecond*10000)
}
