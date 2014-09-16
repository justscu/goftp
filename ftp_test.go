package main

import (
	"github.com/ftp"
	"fmt"
	"io/ioutil"
	"time"
	"os"
	"bytes"
)

// 列举目录下所有文件
func ListAllFiles(conn *ftp.ServerConn, path string) error{
	entrys := make([]*ftp.Entry, 1024)
	entrys, err := conn.List(path)
	if err != nil {
		return err
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
		
		if entrys[i].Type == ftp.EntryTypeFolder {
			ListAllFiles(conn, filename)
		}
	}
	
	return nil
}

// 判断本地文件是否存在
func isFileExist(name string) bool {
	_, err := os.Stat(name)
	return err == nil || os.IsExist(err)
}
// 在本地创建文件
func createFile(name string) bool {
	if isFileExist(name) {
		err := os.Rename(name, name + "." + fmt.Sprintf("%d", time.Now().Nanosecond()))
		if err != nil {
			fmt.Printf("Rename old file[%s] failed[err] \n", name, err)
			return false
		}
	}
	
	fs, err := os.Create(name)
	if err != nil {
		fmt.Printf("create file[%s] failed[err] \n", name, err)
		return false
	}
	
	fs.Close()
	return true
}

// 从服务器上取文件
func GetFileFromServer(conn *ftp.ServerConn, srcName string, dstName string) error{
	r, err := conn.Retr(srcName)
	defer r.Close()
	
	if err != nil {
		return err
	} else {
		buf, err := ioutil.ReadAll(r)
		if err != nil {
			return err
		} else {
			createFile(dstName)
			ioutil.WriteFile(dstName, buf, os.ModeAppend)
//			fmt.Printf("\n%s\n", string(buf))
		}
	}
	
	return nil
}

// 将文件放到服务器上
func PutFileToServer(conn *ftp.ServerConn, srcName string, dstName string) error {
	if isFileExist(srcName) == false {
		fmt.Printf("srcName[%s] not exist \n", srcName)
		return os.ErrNotExist
	}
	
	b, err := ioutil.ReadFile(srcName)
	if err != nil {
		fmt.Printf("read srcName[%s] failed[%s] \n", srcName, err)
		return err
	}
	
	err = conn.Stor(dstName, bytes.NewBufferString(string(b)))
	if err != nil {
		fmt.Printf("stor srcName[%s] dstName[%s] failed[%s] \n", srcName, dstName, err)
	}
	return err
}

func main() {
	fmt.Printf("main \n")
	conn, err := ftp.Connect("10.15.107.74:21")
	if err != nil {
		fmt.Printf("ftp.Connect [%s] \n", err)
		return
	}
	
	err = conn.Login("username", "passwd")
	if err != nil {
		fmt.Printf("conn.Login [%s] \n", err)
		return
	}
	
	dir, err := conn.CurrentDir()
	fmt.Printf("current dir [%s] \n", dir)
	
	//列举目录下所有文件
	ListAllFiles(conn, dir)
	
	// 获取文件内容
	srcFile := "/home/xxx/project/producer/main.cpp"
	dstFile := "/home/ll/main.cpp"
	err = GetFileFromServer(conn, srcFile, dstFile)
	fmt.Printf("GetFileFromServer srcFile[%s] [%s] \n", srcFile, err)
	
	srcFile = "/home/ll/pushproxy.info.2014-09-16.log"
	dstFile = "/home/xxx/tst/main.log"
	// 将文件放到服务器上
	err = PutFileToServer(conn, srcFile, dstFile)
	fmt.Printf("putFileToServer srcFile[%s] [%s] \n", srcFile, err)
	
	fmt.Printf("end \n")
	
	time.Sleep(time.Millisecond*10000)
}