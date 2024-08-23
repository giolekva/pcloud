package main

import (
	"fmt"
	"io"
	"os"
)

type RecordStore interface {
	Log() error
	Add(entry, txt string) error
	AddARecord(entry, ip string) error
	Delete(entry, txt string) error
	DeleteARecord(entry, ip string) error
}

type fsRecordStore struct {
	zone     string
	publicIP []string
	fs       FS
	db       string
}

func (s *fsRecordStore) Log() error {
	r, err := s.fs.Reader(s.db)
	if err != nil {
		return err
	}
	defer r.Close()
	_, err = io.Copy(os.Stdout, r)
	return err
}

func (s *fsRecordStore) read() (*RecordsFile, error) {
	r, err := s.fs.Reader(s.db)
	if err != nil {
		return nil, err
	}
	defer r.Close()
	return NewRecordsFile(io.TeeReader(r, os.Stdout))
}

func (s *fsRecordStore) write(z *RecordsFile) error {
	w, err := s.fs.Writer(s.db)
	if err != nil {
		return err
	}
	defer w.Close()
	return z.Write(io.MultiWriter(w, os.Stdout))
}

func (s *fsRecordStore) Add(entry, txt string) error {
	z, err := s.read()
	if err != nil {
		return err
	}
	fqdn := fmt.Sprintf("%s.%s.", entry, s.zone)
	z.CreateOrReplaceTxtRecord(fqdn, txt)
	// for _, ip := range s.publicIP {
	// 	z.CreateARecord(fqdn, ip)
	// }
	return s.write(z)
}

func (s *fsRecordStore) AddARecord(entry, ip string) error {
	z, err := s.read()
	if err != nil {
		return err
	}
	fqdn := fmt.Sprintf("%s.%s.", entry, s.zone)
	z.CreateARecord(fqdn, ip)
	// for _, ip := range s.publicIP {
	// 	z.CreateARecord(fqdn, ip)
	// }
	return s.write(z)
}

func (s *fsRecordStore) Delete(entry, txt string) error {
	z, err := s.read()
	if err != nil {
		return err
	}
	fqdn := fmt.Sprintf("%s.%s.", entry, s.zone)
	z.DeleteTxtRecord(fqdn, txt)
	// z.DeleteRecordsFor(fqdn)
	return s.write(z)
}

func (s *fsRecordStore) DeleteARecord(entry, ip string) error {
	z, err := s.read()
	if err != nil {
		return err
	}
	fqdn := fmt.Sprintf("%s.%s.", entry, s.zone)
	if err := z.DeleteARecord(fqdn, ip); err != nil {
		return err
	}
	// z.DeleteRecordsFor(fqdn)
	return s.write(z)
}
