package goftp

import (
	"errors"
	"net"
	"os"
)

type File struct {
	Path   string
	client *Client
	pconn  *persistentConn
	conn   net.Conn
	flag   OpenFileFlag
}

func (f *File) Write(bt []byte) (int, error) {
	if f.flag == OPEN_FILE_READ {
		return 0, errors.New("cannot write")
	}
	return f.conn.Write(bt)
}

func (f *File) Read(bt []byte) (int, error) {
	if f.flag == OPEN_FILE_WRITE {
		return 0, errors.New("cannot read")
	}
	return f.conn.Read(bt)
}

func (f *File) Close() error {
	defer f.client.returnConn(f.pconn)
	err := f.conn.Close()
	if err != nil {
		return err
	}
	_, _, err = f.pconn.readResponse()
	if err != nil {
		return err
	}
	return nil
}

type OpenFileFlag = int

const (
	OPEN_FILE_READ  OpenFileFlag = os.O_RDONLY
	OPEN_FILE_WRITE OpenFileFlag = os.O_WRONLY
)

func (c *Client) OpenFile(path string, f OpenFileFlag) (*File, error) {
	pconn, err := c.getIdleConn()
	if err != nil {
		return nil, err
	}

	if err = pconn.setType("I"); err != nil {
		return nil, err
	}

	connGetter, err := pconn.prepareDataConn()
	if err != nil {
		pconn.debug("error preparing data connection: %s", err)
		return nil, err
	}

	var cmd string
	if f == OPEN_FILE_WRITE {
		cmd = "STOR"
	} else if f == OPEN_FILE_READ {
		cmd = "RETR"
	} else {
		panic("this shouldn't happen")
	}

	err = pconn.sendCommandExpected(replyGroupPreliminaryReply, "%s %s", cmd, path)
	if err != nil {
		return nil, err
	}

	dc, err := connGetter()
	if err != nil {
		pconn.debug("error getting data connection: %s", err)
		return nil, err
	}

	file := File{
		Path:   path,
		pconn:  pconn,
		client: c,
		conn:   dc,
		flag:   f,
	}
	return &file, nil
}
