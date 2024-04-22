package main

import (
	"fmt"
)

type RecordStore interface {
	Add(entry, txt string) error
	Delete(entry, txt string) error
}

type fsRecordStore struct {
	zone     string
	publicIP []string
	fs       FS
	db       string
}

func (s *fsRecordStore) read() (*RecordsFile, error) {
	r, err := s.fs.Reader(s.db)
	if err != nil {
		return nil, err
	}
	defer r.Close()
	return NewRecordsFile(r)
}
func (s *fsRecordStore) write(z *RecordsFile) error {
	w, err := s.fs.Writer(s.db)
	if err != nil {
		return err
	}
	defer w.Close()
	return z.Write(w)
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
