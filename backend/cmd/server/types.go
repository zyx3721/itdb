package server

import (
	"database/sql"
	"errors"
	"itdb-backend/cmd/common/primitives"
	"strings"
)

type authLoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Mode     string `json:"mode"`
}

type authLoginResponse struct {
	Token string      `json:"token"`
	User  SessionUser `json:"user"`
}

type itemPayload struct {
	Label            string  `json:"label"`
	ItemTypeID       int64   `json:"itemTypeId"`
	Function         string  `json:"function"`
	ManufacturerID   int64   `json:"manufacturerId"`
	WarrInfo         string  `json:"warrInfo"`
	Model            string  `json:"model"`
	SN               string  `json:"sn"`
	SN2              string  `json:"sn2"`
	SN3              string  `json:"sn3"`
	Origin           string  `json:"origin"`
	WarrantyMonths   *int64  `json:"warrantyMonths"`
	PurchaseDate     string  `json:"purchaseDate"`
	PurchPrice       string  `json:"purchPrice"`
	DNSName          string  `json:"dnsName"`
	DptID            *int64  `json:"dptId"`
	Principal        string  `json:"principal"`
	LocationID       *int64  `json:"locationId"`
	LocAreaID        *int64  `json:"locAreaId"`
	UserID           *int64  `json:"userId"`
	MaintenanceInfo  string  `json:"maintenanceInfo"`
	Comments         string  `json:"comments"`
	IsPart           int64   `json:"isPart"`
	RackID           *int64  `json:"rackId"`
	RackPosition     *int64  `json:"rackPosition"`
	RackPosDepth     *int64  `json:"rackPosDepth"`
	RackMountable    int64   `json:"rackMountable"`
	USize            *int64  `json:"uSize"`
	Status           int64   `json:"status"`
	MACs             string  `json:"macs"`
	IPv4             string  `json:"ipv4"`
	IPv6             string  `json:"ipv6"`
	RemAdmIP         string  `json:"remAdmIp"`
	HD               string  `json:"hd"`
	CPU              string  `json:"cpu"`
	CPUNo            string  `json:"cpuNo"`
	CoresPerCPU      string  `json:"coresPerCpu"`
	RAM              string  `json:"ram"`
	Raid             string  `json:"raid"`
	RaidConfig       string  `json:"raidConfig"`
	PanelPort        string  `json:"panelPort"`
	SwitchID         *int64  `json:"switchId"`
	SwitchPort       string  `json:"switchPort"`
	Ports            string  `json:"ports"`
	ItemLinks        []int64 `json:"itemLinks"`
	InvoiceLinks     []int64 `json:"invoiceLinks"`
	SoftwareLinks    []int64 `json:"softwareLinks"`
	ContractLinks    []int64 `json:"contractLinks"`
	FileLinks        []int64 `json:"fileLinks"`
	CleanupFileLinks []int64 `json:"cleanupFileLinks"`
}

type softwarePayload struct {
	InvoiceID        *int64  `json:"invoiceId"`
	SLicenseInfo     string  `json:"slicenseInfo"`
	Manufacturer     int64   `json:"manufacturerId"`
	Title            string  `json:"title"`
	Version          string  `json:"version"`
	Info             string  `json:"info"`
	PurchaseDate     string  `json:"purchaseDate"`
	LicenseQty       int64   `json:"licenseQty"`
	LicenseType      int64   `json:"licenseType"`
	ItemLinks        []int64 `json:"itemLinks"`
	InvoiceLinks     []int64 `json:"invoiceLinks"`
	ContractLinks    []int64 `json:"contractLinks"`
	FileLinks        []int64 `json:"fileLinks"`
	CleanupFileLinks []int64 `json:"cleanupFileLinks"`
}

type invoicePayload struct {
	VendorID         int64   `json:"vendorId"`
	BuyerID          int64   `json:"buyerId"`
	Number           string  `json:"number"`
	Description      string  `json:"description"`
	Date             string  `json:"date"`
	ItemLinks        []int64 `json:"itemLinks"`
	SoftwareLinks    []int64 `json:"softwareLinks"`
	ContractLinks    []int64 `json:"contractLinks"`
	FileLinks        []int64 `json:"fileLinks"`
	CleanupFileLinks []int64 `json:"cleanupFileLinks"`
}

type contractPayload struct {
	TypeID           int64   `json:"typeId"`
	SubTypeID        int64   `json:"subTypeId"`
	ParentID         *int64  `json:"parentId"`
	Title            string  `json:"title"`
	Number           string  `json:"number"`
	Description      string  `json:"description"`
	Comments         string  `json:"comments"`
	TotalCost        string  `json:"totalCost"`
	ContractorID     int64   `json:"contractorId"`
	StartDate        string  `json:"startDate"`
	CurrentEnd       string  `json:"currentEndDate"`
	Renewals         string  `json:"renewals"`
	ItemLinks        []int64 `json:"itemLinks"`
	SoftwareLinks    []int64 `json:"softwareLinks"`
	InvoiceLinks     []int64 `json:"invoiceLinks"`
	FileLinks        []int64 `json:"fileLinks"`
	CleanupFileLinks []int64 `json:"cleanupFileLinks"`
}

type agentContact struct {
	Name     string `json:"name"`
	Phones   string `json:"phones"`
	Email    string `json:"email"`
	Role     string `json:"role"`
	Comments string `json:"comments"`
}

type agentURL struct {
	Description string `json:"description"`
	URL         string `json:"url"`
}

type agentPayload struct {
	Types       []int64        `json:"types"`
	Type        *int64         `json:"type"`
	Title       string         `json:"title"`
	ContactInfo string         `json:"contactInfo"`
	Contacts    []agentContact `json:"contacts"`
	URLs        []agentURL     `json:"urls"`
}

type userPayload struct {
	Username string `json:"username"`
	UserDesc string `json:"userDesc"`
	Password string `json:"password"`
	UserType int64  `json:"userType"`
}

type locationPayload struct {
	Name  string `json:"name"`
	Floor string `json:"floor"`
}

type rackPayload struct {
	LocationID int64  `json:"locationId"`
	LocAreaID  *int64 `json:"locAreaId"`
	USize      int64  `json:"uSize"`
	RevNums    int64  `json:"revNums"`
	Depth      int64  `json:"depth"`
	Comments   string `json:"comments"`
	Model      string `json:"model"`
	Label      string `json:"label"`
}

type tagMutationPayload struct {
	Name   string `json:"name"`
	Action string `json:"action"`
}

type contractEventPayload struct {
	SiblingID   int64  `json:"siblingId"`
	StartDate   string `json:"startDate"`
	EndDate     string `json:"endDate"`
	Description string `json:"description"`
}

type itemActionPayload struct {
	ActionDate  string `json:"actionDate"`
	Description string `json:"description"`
	InvoiceInfo string `json:"invoiceInfo"`
}

type locAreaPayload struct {
	AreaName string `json:"areaName"`
}

type settingsPayload struct {
	UseLDAP            int64  `json:"useLdap"`
	LDAPServer         string `json:"ldapServer"`
	LDAPDN             string `json:"ldapDn"`
	LDAPBindDN         string `json:"ldapBindDn"`
	LDAPBindPassword   string `json:"ldapBindPassword"`
	LDAPGetUsers       string `json:"ldapGetUsers"`
	LDAPGetUsersFilter string `json:"ldapGetUsersFilter"`
}

func parseDateInput(raw string) (int64, error) {
	return primitives.ParseDateInput(raw)
}

func intParam(v string) (int64, error) {
	return primitives.IntParam(v)
}

func intParamDefault(v string, d int64) int64 {
	return primitives.IntParamDefault(v, d)
}

func listLimitParam(v string, d int64) int64 {
	return primitives.ListLimitParam(v, d)
}

func nullableInt(v *int64) interface{} {
	return primitives.NullableInt(v)
}

func nullableInt64Value(v int64) interface{} {
	return primitives.NullableInt64Value(v)
}

func equalNullInt64(dbValue sql.NullInt64, req *int64) bool {
	return primitives.EqualNullInt64(dbValue, req)
}

func sameDay(ts1, ts2 int64) bool {
	return primitives.SameDay(ts1, ts2)
}

func parseIDCSV(raw string) []int64 {
	return primitives.ParseIDCSV(raw)
}

func validateItem(req itemPayload) error {
	if req.ItemTypeID == 0 {
		return errors.New("itemTypeId is required")
	}
	if req.ManufacturerID == 0 {
		return errors.New("manufacturerId is required")
	}
	if strings.TrimSpace(req.Principal) == "" {
		return errors.New("principal is required")
	}
	if strings.TrimSpace(req.Model) == "" {
		return errors.New("model is required")
	}
	return nil
}

func reqTypeMask(req agentPayload) int64 {
	if req.Type != nil {
		return *req.Type
	}
	var mask int64
	for _, t := range req.Types {
		mask += t
	}
	return mask
}

func sanitizePipeHash(s string) string {
	s = strings.ReplaceAll(s, "|", " ")
	s = strings.ReplaceAll(s, "#", " ")
	return strings.TrimSpace(s)
}

func encodeAgentContacts(contacts []agentContact) string {
	rows := make([]string, 0, len(contacts))
	for _, c := range contacts {
		row := []string{
			sanitizePipeHash(c.Name),
			sanitizePipeHash(c.Phones),
			sanitizePipeHash(c.Email),
			sanitizePipeHash(c.Role),
			sanitizePipeHash(c.Comments),
		}
		rows = append(rows, strings.Join(row, "#"))
	}
	return strings.Join(rows, "|")
}

func encodeAgentURLs(urls []agentURL) string {
	rows := make([]string, 0, len(urls))
	for _, u := range urls {
		row := []string{
			sanitizePipeHash(u.Description),
			sanitizePipeHash(u.URL),
		}
		rows = append(rows, strings.Join(row, "#"))
	}
	return strings.Join(rows, "|")
}

func sanitizeFilename(s string) string {
	return primitives.SanitizeFilename(s)
}

func shortUUID() string {
	return primitives.ShortUUID()
}

func asString(v interface{}) string {
	return primitives.AsString(v)
}

func asInt64(v interface{}) int64 {
	return primitives.AsInt64(v)
}
