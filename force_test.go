package simpleforce

import (
	"log"
	"os"
	"strings"
	"testing"
)

var (
	sfUser  = os.ExpandEnv("${SF_USER}")
	sfPass  = os.ExpandEnv("${SF_PASS}")
	sfToken = os.ExpandEnv("${SF_TOKEN}")
	sfURL   = func() string {
		if os.ExpandEnv("${SF_URL}") != "" {
			return os.ExpandEnv("${SF_URL}")
		} else {
			return DefaultURL
		}
	}()
)

func checkCredentialsAndSkip(t *testing.T) {
	if sfUser == "" || sfPass == "" {
		log.Println(logPrefix, "SF_USER, SF_PASS environment variables are not set.")
		t.Skip()
	}
}

func requireClient(t *testing.T, skippable bool) *Client {
	if skippable {
		checkCredentialsAndSkip(t)
	}

	client := NewClient(sfURL, DefaultClientID, DefaultAPIVersion)
	if client == nil {
		t.Fail()
	}
	err := client.LoginPassword(sfUser, sfPass, sfToken)
	if err != nil {
		t.Fatal()
	}
	return client
}

func TestClient_LoginPassword(t *testing.T) {
	checkCredentialsAndSkip(t)

	client := NewClient(sfURL, DefaultClientID, DefaultAPIVersion)
	if client == nil {
		t.Fatal()
	}

	// Use token
	err := client.LoginPassword(sfUser, sfPass, sfToken)
	if err != nil {
		t.Fail()
	} else {
		log.Println(logPrefix, "sessionID:", client.sessionID)
	}

	err = client.LoginPassword("__INVALID_USER__", "__INVALID_PASS__", "__INVALID_TOKEN__")
	if err == nil {
		t.Fail()
	}
}

func TestClient_LoginPasswordNoToken(t *testing.T) {
	checkCredentialsAndSkip(t)

	client := NewClient(sfURL, DefaultClientID, DefaultAPIVersion)
	if client == nil {
		t.Fatal()
	}

	// Trusted IP must be configured AND the request must be initiated from the trusted IP range.
	err := client.LoginPassword(sfUser, sfPass, "")
	if err != nil {
		t.FailNow()
	} else {
		log.Println(logPrefix, "sessionID:", client.sessionID)
	}
}

func TestClient_LoginOAuth(t *testing.T) {

}

func TestClient_Query(t *testing.T) {
	client := requireClient(t, true)

	q := "SELECT Id,LastModifiedById,LastModifiedDate,ParentId,CommentBody FROM CaseComment"
	result, err := client.Query(q)
	if err != nil {
		log.Println(logPrefix, "query failed,", err)
		t.FailNow()
	}

	log.Println(logPrefix, result.TotalSize, result.Done, result.NextRecordsURL)
	if result.TotalSize < 1 {
		log.Println(logPrefix, "no records returned.")
		t.FailNow()
	}
	for _, record := range result.Records {
		if record.Type() != "CaseComment" {
			t.Fail()
		}
	}
}

func TestClient_Query2(t *testing.T) {
	client := requireClient(t, true)

	q := "Select+id,createdbyid,parentid,parent.casenumber,parent.subject,createdby.name,createdby.alias+from+casecomment"
	result, err := client.Query(q)
	if err != nil {
		t.FailNow()
	}
	if len(result.Records) > 0 {
		comment1 := &result.Records[0]
		case1 := comment1.SObjectField("Case", "Parent").Get()
		if comment1.StringField("ParentId") != case1.ID() {
			t.Fail()
		}
	}
}

func TestClient_Query3(t *testing.T) {
	client := requireClient(t, true)

	q := "SELECT Id FROM CaseComment WHERE CommentBody = 'This comment is created by simpleforce & used for testing'"
	result, err := client.Query(q)
	if err != nil {
		log.Println(logPrefix, "query failed,", err)
		t.FailNow()
	}

	log.Println(logPrefix, result.TotalSize, result.Done, result.NextRecordsURL)
	if result.TotalSize < 1 {
		log.Println(logPrefix, "no records returned.")
		t.FailNow()
	}
	for _, record := range result.Records {
		if record.Type() != "CaseComment" {
			t.Fail()
		}
	}
}

func TestClient_ApexREST(t *testing.T) {
	client := requireClient(t, true)

	endpoint := "services/apexrest/my-custom-endpoint"
	result, err := client.ApexREST(endpoint, "POST", strings.NewReader(`{"my-property": "my-value"}`))
	if err != nil {
		log.Println(logPrefix, "request failed,", err)
		t.FailNow()
	}

	log.Println(logPrefix, string(result))
}

func TestClient_QueryLike(t *testing.T) {
	client := requireClient(t, true)

	q := "Select Id, createdby.name, subject from case where subject like '%simpleforce%'"
	result, err := client.Query(q)
	if err != nil {
		t.FailNow()
	}
	if len(result.Records) > 0 {
		case0 := &result.Records[0]
		if !strings.Contains(case0.StringField("Subject"), "simpleforce") {
			t.FailNow()
		}
	}
}

func TestClient_UploadFileToContentVersion(t *testing.T) {
	client := requireClient(t, true)

	// Create a temporary file
	tmpFile, err := os.CreateTemp("", "simpleforce_testfile_*.txt")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	content := []byte("This is a test file for Salesforce upload.")
	if _, err := tmpFile.Write(content); err != nil {
		t.Fatalf("failed to write to temp file: %v", err)
	}
	tmpFile.Close()

	// Use a valid Salesforce record ID for FirstPublishLocationId (replace with a real one for integration)
	// For test purposes, we use an invalid ID to check error handling
	invalidParentID := "001INVALIDID"
	_, _, err = client.UploadFileToContentVersion(tmpFile.Name(), invalidParentID)
	if err == nil {
		t.Error("expected error for invalid parent ID, got nil")
	}

	// Skip actual upload if no valid parent ID is available
	validParentID := os.Getenv("SF_TEST_PARENT_ID")
	if validParentID == "" {
		t.Skip("SF_TEST_PARENT_ID not set; skipping real upload test")
	}

	cvID, cdID, err := client.UploadFileToContentVersion(
		tmpFile.Name(),
		validParentID,
		WithTitle("Test File"),
		WithDescription("Uploaded by simpleforce test"),
	)
	if err != nil {
		t.Fatalf("upload failed: %v", err)
	}
	if cvID == "" || cdID == "" {
		t.Errorf("expected non-empty IDs, got cvID=%q, cdID=%q", cvID, cdID)
	}
}

func TestClient_DownloadLegacyFile(t *testing.T) {
	client := requireClient(t, true)

	// Test with an invalid Attachment ID (should error)
	invalidAttachmentID := "00PINVALIDID"
	tmpFile, err := os.CreateTemp("", "simpleforce_legacyfile_*.bin")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	tmpFilePath := tmpFile.Name()
	tmpFile.Close()
	defer os.Remove(tmpFilePath)

	err = client.DownloadLegacyFile(invalidAttachmentID, tmpFilePath)
	if err == nil {
		t.Error("expected error for invalid attachment ID, got nil")
	}

	// Test with a valid Attachment ID if provided
	validAttachmentID := os.Getenv("SF_TEST_ATTACHMENT_ID")
	if validAttachmentID == "" {
		t.Skip("SF_TEST_ATTACHMENT_ID not set; skipping real legacy file download test")
	}

	realTmpFile, err := os.CreateTemp("", "simpleforce_legacyfile_real_*.bin")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	realTmpFilePath := realTmpFile.Name()
	realTmpFile.Close()
	defer os.Remove(realTmpFilePath)

	err = client.DownloadLegacyFile(validAttachmentID, realTmpFilePath)
	if err != nil {
		t.Fatalf("download failed: %v", err)
	}

	// Check that the file exists and is non-empty
	info, err := os.Stat(realTmpFilePath)
	if err != nil {
		t.Fatalf("downloaded file not found: %v", err)
	}
	if info.Size() == 0 {
		t.Error("downloaded file is empty")
	}
}

func TestMain(m *testing.M) {
	m.Run()
}
