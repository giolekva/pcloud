package main

type UIEvent interface{}

type EventScanBarcode struct{}

type EventGetInviteQRCode struct{}

type EventGetJoinQRCode struct{}

type EventApproveOther struct{}
