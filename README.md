### goftp

使用`github.com/jlaffaye/ftp`基础库，并对基本功能进行包装。`github.com/jlaffaye/ftp`库，会不定期的fix一些bug，所以在使用时，最好更新到最新版本。

在实际使用过程中，我们发现当ftp-server是pure-ftpd时，会有`EPSV`问题。go-ftp客户端首先会向server发送`FEAT`命令，获取服务器支持的一些命令，返回的有'EPSV'，但当go-ftp客户端使用`openDataConn`去与服务器建立数据连接时，会报错（不支持EPSV）。该bug没有在vsftpd服务器中发现。

简单的修正方法，就是注视掉`openDataConn`中的EPSV选项，直接使用`PASV`模式。见`jlaffaye/ftp/ftp.go:217`行。
