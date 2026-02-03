package library

import (
	"github.com/simpledms/simpledms/model/common/attributetype"
	"github.com/simpledms/simpledms/model/common/fieldtype"
	"github.com/simpledms/simpledms/model/tagging/tagtype"
	wx "github.com/simpledms/simpledms/ui/widget"
)

type BuiltinTemplate struct {
	Key        string
	Name       string
	Icon       string
	Fields     []BuiltinField
	Tags       []BuiltinTag
	Attributes []BuiltinAttribute
}

type BuiltinField struct {
	Key  string
	Name string
	Type fieldtype.FieldType
	Unit string
}

type BuiltinTag struct {
	Key      string
	Name     string
	Type     tagtype.TagType
	GroupKey string
	Color    string
	Icon     string
}

type BuiltinAttribute struct {
	Name         string
	Type         attributetype.AttributeType
	FieldKey     string
	TagKey       string
	IsRequired   bool
	IsNameGiving bool
}

var builtinTemplates = []BuiltinTemplate{
	{
		Key:  "invoice",
		Name: "Invoice",
		Icon: "receipt_long",
		Fields: []BuiltinField{
			{Key: "invoice_number", Name: "Invoice number", Type: fieldtype.Text},
			{Key: "invoice_date", Name: "Invoice date", Type: fieldtype.Date},
			{Key: "total_amount", Name: "Total amount", Type: fieldtype.Money},
			{Key: "due_date", Name: "Due date", Type: fieldtype.Date},
		},
		Tags: []BuiltinTag{
			{Key: "invoice_status", Name: "Invoice status", Type: tagtype.Group},
			{Key: "invoice_status_open", Name: "Open", Type: tagtype.Simple, GroupKey: "invoice_status"},
			{Key: "invoice_status_paid", Name: "Paid", Type: tagtype.Simple, GroupKey: "invoice_status"},
			{Key: "invoice_status_overdue", Name: "Overdue", Type: tagtype.Simple, GroupKey: "invoice_status"},
			{Key: "currency", Name: "Currency", Type: tagtype.Group},
			{Key: "currency_eur", Name: "EUR", Type: tagtype.Simple, GroupKey: "currency"},
			{Key: "currency_usd", Name: "USD", Type: tagtype.Simple, GroupKey: "currency"},
			{Key: "currency_gbp", Name: "GBP", Type: tagtype.Simple, GroupKey: "currency"},
			{Key: "organization", Name: "Organization", Type: tagtype.Group},
		},
		Attributes: []BuiltinAttribute{
			{Type: attributetype.Field, FieldKey: "invoice_number", IsNameGiving: true},
			{Type: attributetype.Field, FieldKey: "invoice_date"},
			{Type: attributetype.Field, FieldKey: "total_amount"},
			{Type: attributetype.Field, FieldKey: "due_date"},
			{Type: attributetype.Tag, TagKey: "invoice_status", Name: "Status", IsNameGiving: true},
			{Type: attributetype.Tag, TagKey: "currency", Name: "Currency"},
			{Type: attributetype.Tag, TagKey: "organization", Name: "Supplier", IsNameGiving: true},
		},
	},
	{
		Key:  "receipt",
		Name: "Receipt",
		Icon: "receipt",
		Fields: []BuiltinField{
			{Key: "receipt_date", Name: "Receipt date", Type: fieldtype.Date},
			{Key: "receipt_total", Name: "Total amount", Type: fieldtype.Money},
			{Key: "payment_method", Name: "Payment method", Type: fieldtype.Text},
		},
		Tags: []BuiltinTag{
			{Key: "expense_category", Name: "Receipt category", Type: tagtype.Group},
			{Key: "expense_category_travel", Name: "Travel", Type: tagtype.Simple, GroupKey: "expense_category"},
			{Key: "expense_category_meals", Name: "Meals", Type: tagtype.Simple, GroupKey: "expense_category"},
			{Key: "expense_category_office", Name: "Office", Type: tagtype.Simple, GroupKey: "expense_category"},
			{Key: "expense_category_software", Name: "Software", Type: tagtype.Simple, GroupKey: "expense_category"},
			{Key: "expense_category_other", Name: "Other", Type: tagtype.Simple, GroupKey: "expense_category"},
			{Key: "organization", Name: "Organization", Type: tagtype.Group},
		},
		Attributes: []BuiltinAttribute{
			{Type: attributetype.Tag, TagKey: "organization", Name: "Vendor", IsNameGiving: true},
			{Type: attributetype.Field, FieldKey: "receipt_date"},
			{Type: attributetype.Field, FieldKey: "receipt_total"},
			{Type: attributetype.Field, FieldKey: "payment_method"},
			{Type: attributetype.Tag, TagKey: "expense_category", Name: "Category"},
		},
	},
	{
		Key:  "contract",
		Name: "Contract",
		Icon: "gavel",
		Fields: []BuiltinField{
			{Key: "contract_name", Name: "Contract name", Type: fieldtype.Text},
			{Key: "start_date", Name: "Start date", Type: fieldtype.Date},
			{Key: "end_date", Name: "End date", Type: fieldtype.Date},
			{Key: "contract_value", Name: "Value", Type: fieldtype.Money},
		},
		Tags: []BuiltinTag{
			{Key: "contract_status", Name: "Contract status", Type: tagtype.Group},
			{Key: "contract_status_draft", Name: "Draft", Type: tagtype.Simple, GroupKey: "contract_status"},
			{Key: "contract_status_active", Name: "Active", Type: tagtype.Simple, GroupKey: "contract_status"},
			{Key: "contract_status_expired", Name: "Expired", Type: tagtype.Simple, GroupKey: "contract_status"},
			{Key: "contract_status_terminated", Name: "Terminated", Type: tagtype.Simple, GroupKey: "contract_status"},
			{Key: "contract_type", Name: "Contract type", Type: tagtype.Group},
			{Key: "contract_type_service", Name: "Service", Type: tagtype.Simple, GroupKey: "contract_type"},
			{Key: "contract_type_sales", Name: "Sales", Type: tagtype.Simple, GroupKey: "contract_type"},
			{Key: "contract_type_nda", Name: "NDA", Type: tagtype.Simple, GroupKey: "contract_type"},
			{Key: "contract_type_other", Name: "Other", Type: tagtype.Simple, GroupKey: "contract_type"},
			{Key: "organization", Name: "Organization", Type: tagtype.Group},
		},
		Attributes: []BuiltinAttribute{
			{Type: attributetype.Field, FieldKey: "contract_name", IsNameGiving: true},
			{Type: attributetype.Field, FieldKey: "start_date"},
			{Type: attributetype.Field, FieldKey: "end_date"},
			{Type: attributetype.Field, FieldKey: "contract_value"},
			{Type: attributetype.Tag, TagKey: "contract_status", Name: "Status", IsNameGiving: true},
			{Type: attributetype.Tag, TagKey: "contract_type", Name: "Type"},
			{Type: attributetype.Tag, TagKey: "organization", Name: "Parties", IsNameGiving: true},
		},
	},
	{
		Key:  "purchase_order",
		Name: "Purchase Order",
		Icon: "shopping_cart",
		Fields: []BuiltinField{
			{Key: "po_number", Name: "PO number", Type: fieldtype.Text},
			{Key: "order_date", Name: "Order date", Type: fieldtype.Date},
			{Key: "total_amount", Name: "Total amount", Type: fieldtype.Money},
		},
		Tags: []BuiltinTag{
			{Key: "po_status", Name: "Purchase order status", Type: tagtype.Group},
			{Key: "po_status_draft", Name: "Draft", Type: tagtype.Simple, GroupKey: "po_status"},
			{Key: "po_status_sent", Name: "Sent", Type: tagtype.Simple, GroupKey: "po_status"},
			{Key: "po_status_approved", Name: "Approved", Type: tagtype.Simple, GroupKey: "po_status"},
			{Key: "po_status_fulfilled", Name: "Fulfilled", Type: tagtype.Simple, GroupKey: "po_status"},
			{Key: "organization", Name: "Organization", Type: tagtype.Group},
			{Key: "person", Name: "Person", Type: tagtype.Group},
		},
		Attributes: []BuiltinAttribute{
			{Type: attributetype.Field, FieldKey: "po_number", IsNameGiving: true},
			{Type: attributetype.Field, FieldKey: "order_date"},
			{Type: attributetype.Field, FieldKey: "total_amount"},
			{Type: attributetype.Tag, TagKey: "po_status", Name: "Status", IsNameGiving: true},
			{Type: attributetype.Tag, TagKey: "organization", Name: "Supplier", IsNameGiving: true},
			{Type: attributetype.Tag, TagKey: "person", Name: "Requested by"},
		},
	},
	{
		Key:  "policy",
		Name: "Policy",
		Icon: "policy",
		Fields: []BuiltinField{
			{Key: "policy_name", Name: "Policy name", Type: fieldtype.Text},
			{Key: "effective_date", Name: "Effective date", Type: fieldtype.Date},
			{Key: "version", Name: "Version", Type: fieldtype.Text},
		},
		Tags: []BuiltinTag{
			{Key: "policy_status", Name: "Policy status", Type: tagtype.Group},
			{Key: "policy_status_draft", Name: "Draft", Type: tagtype.Simple, GroupKey: "policy_status"},
			{Key: "policy_status_active", Name: "Active", Type: tagtype.Simple, GroupKey: "policy_status"},
			{Key: "policy_status_archived", Name: "Archived", Type: tagtype.Simple, GroupKey: "policy_status"},
			{Key: "department", Name: "Policy department", Type: tagtype.Group},
			{Key: "department_hr", Name: "HR", Type: tagtype.Simple, GroupKey: "department"},
			{Key: "department_it", Name: "IT", Type: tagtype.Simple, GroupKey: "department"},
			{Key: "department_finance", Name: "Finance", Type: tagtype.Simple, GroupKey: "department"},
			{Key: "department_operations", Name: "Operations", Type: tagtype.Simple, GroupKey: "department"},
			{Key: "department_other", Name: "Other", Type: tagtype.Simple, GroupKey: "department"},
			{Key: "person", Name: "Person", Type: tagtype.Group},
		},
		Attributes: []BuiltinAttribute{
			{Type: attributetype.Field, FieldKey: "policy_name", IsNameGiving: true},
			{Type: attributetype.Field, FieldKey: "effective_date"},
			{Type: attributetype.Field, FieldKey: "version"},
			{Type: attributetype.Tag, TagKey: "policy_status", Name: "Status", IsNameGiving: true},
			{Type: attributetype.Tag, TagKey: "department", Name: "Department"},
			{Type: attributetype.Tag, TagKey: "person", Name: "Owner", IsNameGiving: true},
		},
	},
	{
		Key:  "project_document",
		Name: "Project Document",
		Icon: "assignment",
		Fields: []BuiltinField{
			{Key: "project_name", Name: "Project name", Type: fieldtype.Text},
			{Key: "start_date", Name: "Start date", Type: fieldtype.Date},
			{Key: "target_date", Name: "Target date", Type: fieldtype.Date},
		},
		Tags: []BuiltinTag{
			{Key: "project_status", Name: "Project status", Type: tagtype.Group},
			{Key: "project_status_draft", Name: "Draft", Type: tagtype.Simple, GroupKey: "project_status"},
			{Key: "project_status_in_progress", Name: "In progress", Type: tagtype.Simple, GroupKey: "project_status"},
			{Key: "project_status_complete", Name: "Complete", Type: tagtype.Simple, GroupKey: "project_status"},
			{Key: "project_status_on_hold", Name: "On hold", Type: tagtype.Simple, GroupKey: "project_status"},
			{Key: "project_type", Name: "Project type", Type: tagtype.Group},
			{Key: "project_type_plan", Name: "Plan", Type: tagtype.Simple, GroupKey: "project_type"},
			{Key: "project_type_report", Name: "Report", Type: tagtype.Simple, GroupKey: "project_type"},
			{Key: "project_type_spec", Name: "Spec", Type: tagtype.Simple, GroupKey: "project_type"},
			{Key: "project_type_other", Name: "Other", Type: tagtype.Simple, GroupKey: "project_type"},
			{Key: "person", Name: "Person", Type: tagtype.Group},
		},
		Attributes: []BuiltinAttribute{
			{Type: attributetype.Field, FieldKey: "project_name", IsNameGiving: true},
			{Type: attributetype.Field, FieldKey: "start_date"},
			{Type: attributetype.Field, FieldKey: "target_date"},
			{Type: attributetype.Tag, TagKey: "project_status", Name: "Status", IsNameGiving: true},
			{Type: attributetype.Tag, TagKey: "project_type", Name: "Type"},
			{Type: attributetype.Tag, TagKey: "person", Name: "Owner", IsNameGiving: true},
		},
	},
	{
		Key:  "meeting_notes",
		Name: "Meeting Notes",
		Icon: "event_note",
		Fields: []BuiltinField{
			{Key: "meeting_date", Name: "Meeting date", Type: fieldtype.Date},
			{Key: "title", Name: "Title", Type: fieldtype.Text},
		},
		Tags: []BuiltinTag{
			{Key: "meeting_type", Name: "Meeting type", Type: tagtype.Group},
			{Key: "meeting_type_internal", Name: "Internal", Type: tagtype.Simple, GroupKey: "meeting_type"},
			{Key: "meeting_type_client", Name: "Client", Type: tagtype.Simple, GroupKey: "meeting_type"},
			{Key: "meeting_type_vendor", Name: "Vendor", Type: tagtype.Simple, GroupKey: "meeting_type"},
			{Key: "meeting_status", Name: "Meeting status", Type: tagtype.Group},
			{Key: "meeting_status_draft", Name: "Draft", Type: tagtype.Simple, GroupKey: "meeting_status"},
			{Key: "meeting_status_final", Name: "Final", Type: tagtype.Simple, GroupKey: "meeting_status"},
			{Key: "person", Name: "Person", Type: tagtype.Group},
		},
		Attributes: []BuiltinAttribute{
			{Type: attributetype.Field, FieldKey: "title", IsNameGiving: true},
			{Type: attributetype.Field, FieldKey: "meeting_date"},
			{Type: attributetype.Tag, TagKey: "meeting_type", Name: "Type"},
			{Type: attributetype.Tag, TagKey: "meeting_status", Name: "Status", IsNameGiving: true},
			{Type: attributetype.Tag, TagKey: "person", Name: "Organizer", IsNameGiving: true},
			{Type: attributetype.Tag, TagKey: "person", Name: "Participants", IsNameGiving: true},
		},
	},
	{
		Key:  "expense_report",
		Name: "Expense Report",
		Icon: "account_balance_wallet",
		Fields: []BuiltinField{
			{Key: "report_period", Name: "Report period", Type: fieldtype.Text},
			{Key: "total_amount", Name: "Total amount", Type: fieldtype.Money},
			{Key: "submission_date", Name: "Submission date", Type: fieldtype.Date},
		},
		Tags: []BuiltinTag{
			{Key: "expense_status", Name: "Expense status", Type: tagtype.Group},
			{Key: "expense_status_draft", Name: "Draft", Type: tagtype.Simple, GroupKey: "expense_status"},
			{Key: "expense_status_submitted", Name: "Submitted", Type: tagtype.Simple, GroupKey: "expense_status"},
			{Key: "expense_status_approved", Name: "Approved", Type: tagtype.Simple, GroupKey: "expense_status"},
			{Key: "expense_status_rejected", Name: "Rejected", Type: tagtype.Simple, GroupKey: "expense_status"},
			{Key: "department", Name: "Expense department", Type: tagtype.Group},
			{Key: "department_hr", Name: "HR", Type: tagtype.Simple, GroupKey: "department"},
			{Key: "department_it", Name: "IT", Type: tagtype.Simple, GroupKey: "department"},
			{Key: "department_finance", Name: "Finance", Type: tagtype.Simple, GroupKey: "department"},
			{Key: "department_operations", Name: "Operations", Type: tagtype.Simple, GroupKey: "department"},
			{Key: "department_other", Name: "Other", Type: tagtype.Simple, GroupKey: "department"},
			{Key: "person", Name: "Person", Type: tagtype.Group},
		},
		Attributes: []BuiltinAttribute{
			{Type: attributetype.Tag, TagKey: "person", Name: "Employee", IsNameGiving: true},
			{Type: attributetype.Field, FieldKey: "report_period", IsNameGiving: true},
			{Type: attributetype.Field, FieldKey: "total_amount"},
			{Type: attributetype.Field, FieldKey: "submission_date"},
			{Type: attributetype.Tag, TagKey: "expense_status", Name: "Status", IsNameGiving: true},
			{Type: attributetype.Tag, TagKey: "department", Name: "Department"},
		},
	},
}

func BuiltinTemplates() []BuiltinTemplate {
	return builtinTemplates
}

func init() {
	registerBuiltinStrings()
}

// for detection by `go text`
func registerBuiltinStrings() {
	wx.T("Invoice")
	wx.T("Invoice number")
	wx.T("Invoice date")
	wx.T("Supplier")
	wx.T("Total amount")
	wx.T("Due date")
	wx.T("Status")
	wx.T("Open")
	wx.T("Paid")
	wx.T("Overdue")
	wx.T("Invoice status")
	wx.T("Currency")
	wx.T("EUR")
	wx.T("USD")
	wx.T("GBP")
	wx.T("Organization")
	wx.T("Receipt")
	wx.T("Receipt date")
	wx.T("Vendor")
	wx.T("Payment method")
	wx.T("Receipt category")
	wx.T("Travel")
	wx.T("Meals")
	wx.T("Office")
	wx.T("Software")
	wx.T("Other")
	wx.T("Contract")
	wx.T("Contract name")
	wx.T("Parties")
	wx.T("Start date")
	wx.T("End date")
	wx.T("Value")
	wx.T("Draft")
	wx.T("Active")
	wx.T("Expired")
	wx.T("Terminated")
	wx.T("Contract status")
	wx.T("Contract type")
	wx.T("Type")
	wx.T("Service")
	wx.T("Sales")
	wx.T("NDA")
	wx.T("Purchase Order")
	wx.T("PO number")
	wx.T("Order date")
	wx.T("Requested by")
	wx.T("Sent")
	wx.T("Approved")
	wx.T("Fulfilled")
	wx.T("Purchase order status")
	wx.T("Policy")
	wx.T("Policy name")
	wx.T("Effective date")
	wx.T("Version")
	wx.T("Owner")
	wx.T("Archived")
	wx.T("Policy status")
	wx.T("Policy department")
	wx.T("Department")
	wx.T("HR")
	wx.T("IT")
	wx.T("Finance")
	wx.T("Operations")
	wx.T("Project Document")
	wx.T("Project name")
	wx.T("Target date")
	wx.T("In progress")
	wx.T("Complete")
	wx.T("On hold")
	wx.T("Project status")
	wx.T("Project type")
	wx.T("Plan")
	wx.T("Report")
	wx.T("Spec")
	wx.T("Meeting Notes")
	wx.T("Meeting date")
	wx.T("Title")
	wx.T("Organizer")
	wx.T("Participants")
	wx.T("Internal")
	wx.T("Client")
	wx.T("Final")
	wx.T("Meeting type")
	wx.T("Meeting status")
	wx.T("Expense Report")
	wx.T("Report period")
	wx.T("Employee")
	wx.T("Submission date")
	wx.T("Submitted")
	wx.T("Rejected")
	wx.T("Expense status")
	wx.T("Expense department")
	wx.T("Person")
}
