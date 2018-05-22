// Copyright 2017 The ACH Authors
// Use of this source code is governed by an Apache License
// license that can be found in the LICENSE file.

package ach

import (
	"fmt"
	"strconv"
	"strings"
)

// EntryDetail contains the actual transaction data for an individual entry.
// Fields include those designating the entry as a deposit (credit) or
// withdrawal (debit), the transit routing number for the entry recipient’s financial
// institution, the account number (left justify,no zero fill), name, and dollar amount.
type EntryDetail struct {
	// ID is a client defined string used as a reference to this record.
	ID string `json:"id"`
	// RecordType defines the type of record in the block. 6
	recordType string
	// TransactionCode if the receivers account is:
	// Credit (deposit) to checking account ‘22’
	// Prenote for credit to checking account ‘23’
	// Debit (withdrawal) to checking account ‘27’
	// Prenote for debit to checking account ‘28’
	// Credit to savings account ‘32’
	// Prenote for credit to savings account ‘33’
	// Debit to savings account ‘37’
	// Prenote for debit to savings account ‘38’
	TransactionCode int `json:"transactionCode"`

	// RDFIIdentification is the RDFI's routing number without the last digit.
	// Receiving Depository Financial Institution
	RDFIIdentification string `json:"RDFIIdentification"`

	// CheckDigit the last digit of the RDFI's routing number
	CheckDigit string `json:"checkDigit"`

	// DFIAccountNumber is the receiver's bank account number you are crediting/debiting.
	// It important to note that this is an alphanumeric field, so its space padded, no zero padded
	DFIAccountNumber string `json:"DFIAccountNumber"`

	// Amount Number of cents you are debiting/crediting this account
	Amount int `json:"amount"`

	// IdentificationNumber an internal identification (alphanumeric) that
	// you use to uniquely identify this Entry Detail Record
	IdentificationNumber string `json:"identificationNumber,omitempty"`

	// IndividualName The name of the receiver, usually the name on the bank account
	IndividualName string `json:"individualName"`

	// DiscretionaryData allows ODFIs to include codes, of significance only to them,
	// to enable specialized handling of the entry. There will be no
	// standardized interpretation for the value of this field. It can either
	// be a single two-character code, or two distinct one-character codes,
	// according to the needs of the ODFI and/or Originator involved. This
	// field must be returned intact for any returned entry.
	//
	// WEB uses the Discretionary Data Field as the Payment Type Code
	DiscretionaryData string `json:"discretionaryData,omitempty"`

	// AddendaRecordIndicator indicates the existence of an Addenda Record.
	// A value of "1" indicates that one ore more addenda records follow,
	// and "0" means no such record is present.
	AddendaRecordIndicator int `json:"addendaRecordIndicator,omitempty"`

	// TraceNumber assigned by the ODFI in ascending sequence, is included in each
	// Entry Detail Record, Corporate Entry Detail Record, and addenda Record.
	// Trace Numbers uniquely identify each entry within a batch in an ACH input file.
	// In association with the Batch Number, transmission (File Creation) Date,
	// and File ID Modifier, the Trace Number uniquely identifies an entry within a given file.
	// For addenda Records, the Trace Number will be identical to the Trace Number
	// in the associated Entry Detail Record, since the Trace Number is associated
	// with an entry or item rather than a physical record.
	TraceNumber int `json:"traceNumber,omitempty"`

	// Addendum a list of Addenda for the Entry Detail
	Addendum []Addendumer `json:"addendum,omitempty"`
	// Category defines if the entry is a Forward, Return, or NOC
	Category string `json:"category,omitempty"`
	// validator is composed for data validation
	validator
	// converters is composed for ACH to golang Converters
	converters
}

const (
	// CategoryForward defines the entry as being sent to the receiving institution
	CategoryForward = "Forward"
	// CategoryReturn defines the entry as being a return of a forward entry back to the originating institution
	CategoryReturn = "Return"
	// CategoryNOC defines the entry as being a notification of change of a forward entry to the originating institution
	CategoryNOC = "NOC"
)

// NewEntryDetail returns a new EntryDetail with default values for non exported fields
func NewEntryDetail() *EntryDetail {
	entry := &EntryDetail{
		recordType: "6",
		Category:   CategoryForward,
	}
	return entry
}

// Parse takes the input record string and parses the EntryDetail values
func (ed *EntryDetail) Parse(record string) {
	// 1-1 Always "6"
	ed.recordType = "6"
	// 2-3 is checking credit 22 debit 27 savings credit 32 debit 37
	ed.TransactionCode = ed.parseNumField(record[1:3])
	// 4-11 the RDFI's routing number without the last digit.
	ed.RDFIIdentification = ed.parseStringField(record[3:11])
	// 12-12 The last digit of the RDFI's routing number
	ed.CheckDigit = ed.parseStringField(record[11:12])
	// 13-29 The receiver's bank account number you are crediting/debiting
	ed.DFIAccountNumber = record[12:29]
	// 30-39 Number of cents you are debiting/crediting this account
	ed.Amount = ed.parseNumField(record[29:39])
	// 40-54 An internal identification (alphanumeric) that you use to uniquely identify this Entry Detail Record
	ed.IdentificationNumber = record[39:54]
	// 55-76 The name of the receiver, usually the name on the bank account
	ed.IndividualName = record[54:76]
	// 77-78 allows ODFIs to include codes of significance only to them
	// For WEB transaction this field is the PaymentType which is either R(reoccurring) or S(single)
	// normally blank
	ed.DiscretionaryData = record[76:78]
	// 79-79 1 if addenda exists 0 if it does not
	ed.AddendaRecordIndicator = ed.parseNumField(record[78:79])
	// 80-94 An internal identification (alphanumeric) that you use to uniquely identify
	// this Entry Detail Record This number should be unique to the transaction and will help identify the transaction in case of an inquiry
	ed.TraceNumber = ed.parseNumField(record[79:94])
}

// String writes the EntryDetail struct to a 94 character string.
func (ed *EntryDetail) String() string {
	return fmt.Sprintf("%v%v%v%v%v%v%v%v%v%v%v",
		ed.recordType,
		ed.TransactionCode,
		ed.RDFIIdentificationField(),
		ed.CheckDigit,
		ed.DFIAccountNumberField(),
		ed.AmountField(),
		ed.IdentificationNumberField(),
		ed.IndividualNameField(),
		ed.DiscretionaryDataField(),
		ed.AddendaRecordIndicator,
		ed.TraceNumberField())
}

// Validate performs NACHA format rule checks on the record and returns an error if not Validated
// The first error encountered is returned and stops that parsing.
func (ed *EntryDetail) Validate() error {
	if err := ed.fieldInclusion(); err != nil {
		return err
	}
	if ed.recordType != "6" {
		msg := fmt.Sprintf(msgRecordType, 6)
		return &FieldError{FieldName: "recordType", Value: ed.recordType, Msg: msg}
	}
	if err := ed.isTransactionCode(ed.TransactionCode); err != nil {
		return &FieldError{FieldName: "TransactionCode", Value: strconv.Itoa(ed.TransactionCode), Msg: err.Error()}
	}
	if err := ed.isAlphanumeric(ed.DFIAccountNumber); err != nil {
		return &FieldError{FieldName: "DFIAccountNumber", Value: ed.DFIAccountNumber, Msg: err.Error()}
	}
	if err := ed.isAlphanumeric(ed.IdentificationNumber); err != nil {
		return &FieldError{FieldName: "IdentificationNumber", Value: ed.IdentificationNumber, Msg: err.Error()}
	}
	if err := ed.isAlphanumeric(ed.IndividualName); err != nil {
		return &FieldError{FieldName: "IndividualName", Value: ed.IndividualName, Msg: err.Error()}
	}
	if err := ed.isAlphanumeric(ed.DiscretionaryData); err != nil {
		return &FieldError{FieldName: "DiscretionaryData", Value: ed.DiscretionaryData, Msg: err.Error()}
	}

	calculated := ed.CalculateCheckDigit(ed.RDFIIdentificationField())

	edCheckDigit, err := strconv.Atoi(ed.CheckDigit)
	if err != nil {
		return err
	}

	if calculated != edCheckDigit {
		msg := fmt.Sprintf(msgValidCheckDigit, calculated)
		return &FieldError{FieldName: "RDFIIdentification", Value: ed.CheckDigit, Msg: msg}
	}
	return nil
}

// fieldInclusion validate mandatory fields are not default values. If fields are
// invalid the ACH transfer will be returned.
func (ed *EntryDetail) fieldInclusion() error {
	if ed.recordType == "" {
		return &FieldError{FieldName: "recordType", Value: ed.recordType, Msg: msgFieldInclusion}
	}
	if ed.TransactionCode == 0 {
		return &FieldError{FieldName: "TransactionCode", Value: strconv.Itoa(ed.TransactionCode), Msg: msgFieldInclusion}
	}
	if ed.RDFIIdentification == "" {
		return &FieldError{FieldName: "RDFIIdentification", Value: ed.RDFIIdentificationField(), Msg: msgFieldInclusion}
	}
	if ed.DFIAccountNumber == "" {
		return &FieldError{FieldName: "DFIAccountNumber", Value: ed.DFIAccountNumber, Msg: msgFieldInclusion}
	}
	if ed.IndividualName == "" {
		return &FieldError{FieldName: "IndividualName", Value: ed.IndividualName, Msg: msgFieldInclusion}
	}
	if ed.TraceNumber == 0 {
		return &FieldError{FieldName: "TraceNumber", Value: ed.TraceNumberField(), Msg: msgFieldInclusion}
	}
	return nil
}

// AddAddenda appends an Addendumer to the EntryDetail
func (ed *EntryDetail) AddAddenda(addenda Addendumer) []Addendumer {
	ed.AddendaRecordIndicator = 1
	// checks to make sure that we only have either or, not both
	switch addenda.(type) {
	case *Addenda99:
		ed.Category = CategoryReturn
		ed.Addendum = nil
		ed.Addendum = append(ed.Addendum, addenda)
		return ed.Addendum
	case *Addenda98:
		ed.Category = CategoryNOC
		ed.Addendum = nil
		ed.Addendum = append(ed.Addendum, addenda)
		return ed.Addendum
		// default is current *Addenda05
	default:
		ed.Category = CategoryForward
		ed.Addendum = append(ed.Addendum, addenda)
		return ed.Addendum
	}
}

// SetRDFI takes the 9 digit RDFI account number and separates it for RDFIIdentification and CheckDigit
func (ed *EntryDetail) SetRDFI(rdfi string) *EntryDetail {
	s := ed.stringRTNField(rdfi, 9)
	ed.RDFIIdentification = ed.parseStringField(s[:8])
	ed.CheckDigit = ed.parseStringField(s[8:9])
	return ed
}

// SetTraceNumber takes first 8 digits of ODFI and concatenates a sequence number onto the TraceNumber
func (ed *EntryDetail) SetTraceNumber(ODFIIdentification string, seq int) {
	trace := ed.stringRTNField(ODFIIdentification, 8) + ed.numericField(seq, 7)
	ed.TraceNumber = ed.parseNumField(trace)
}

// RDFIIdentificationField get the rdfiIdentification with zero padding
func (ed *EntryDetail) RDFIIdentificationField() string {
	return ed.stringRTNField(ed.RDFIIdentification, 8)
}

// DFIAccountNumberField gets the DFIAccountNumber with space padding
func (ed *EntryDetail) DFIAccountNumberField() string {
	return ed.alphaField(ed.DFIAccountNumber, 17)
}

// AmountField returns a zero padded string of amount
func (ed *EntryDetail) AmountField() string {
	return ed.numericField(ed.Amount, 10)
}

// IdentificationNumberField returns a space padded string of IdentificationNumber
func (ed *EntryDetail) IdentificationNumberField() string {
	return ed.alphaField(ed.IdentificationNumber, 15)
}

// IndividualNameField returns a space padded string of IndividualName
func (ed *EntryDetail) IndividualNameField() string {
	return ed.alphaField(ed.IndividualName, 22)
}

// ReceivingCompanyField is used in CCD files but returns the underlying IndividualName field
func (ed *EntryDetail) ReceivingCompanyField() string {
	return ed.IndividualNameField()
}

// SetReceivingCompany setter for CCD receiving company individual name
func (ed *EntryDetail) SetReceivingCompany(s string) {
	ed.IndividualName = s
}

// DiscretionaryDataField returns a space padded string of DiscretionaryData
func (ed *EntryDetail) DiscretionaryDataField() string {
	return ed.alphaField(ed.DiscretionaryData, 2)
}

// PaymentTypeField returns the discretionary data field used in WEB batch files
func (ed *EntryDetail) PaymentTypeField() string {
	// because DiscretionaryData can be changed outside of PaymentType we reset the value for safety
	ed.SetPaymentType(ed.DiscretionaryData)
	return ed.DiscretionaryData
}

// SetPaymentType as R (Recurring) all other values will result in S (single)
func (ed *EntryDetail) SetPaymentType(t string) {
	t = strings.ToUpper(strings.TrimSpace(t))
	if t == "R" {
		ed.DiscretionaryData = "R"
	} else {
		ed.DiscretionaryData = "S"
	}
}

// TraceNumberField returns a zero padded traceNumber string
func (ed *EntryDetail) TraceNumberField() string {
	return ed.numericField(ed.TraceNumber, 15)
}

// CreditOrDebit returns a "C" for credit or "D" for debit based on the entry TransactionCode
func (ed *EntryDetail) CreditOrDebit() string {
	tc := strconv.Itoa(ed.TransactionCode)
	// take the second number in the Transaction code
	switch tc[1:2] {
	case "1", "2", "3":
		return "C"
	case "6", "7", "8":
		return "D"
	}
	return ""
}
