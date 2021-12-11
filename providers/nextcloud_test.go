package providers

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
)

const formatJSON = "format=json"

func testNextcloudProvider(hostname string) *NextcloudProvider {
	p := NewNextcloudProvider(
		&ProviderData{
			ProviderName: "",
			LoginURL:     &url.URL{},
			RedeemURL:    &url.URL{},
			ProfileURL:   &url.URL{},
			ValidateURL:  &url.URL{},
			Scope:        ""})
	if hostname != "" {
		updateURL(p.Data().LoginURL, hostname)
		updateURL(p.Data().RedeemURL, hostname)
		updateURL(p.Data().ProfileURL, hostname)
		updateURL(p.Data().ValidateURL, hostname)
	}
	return p
}

func TestNextcloudProviderDefaults(t *testing.T) {
	p := testNextcloudProvider("")
	assert.NotEqual(t, nil, p)
	assert.Equal(t, "Nextcloud", p.Data().ProviderName)
	assert.Equal(t, "",
		p.Data().LoginURL.String())
	assert.Equal(t, "",
		p.Data().RedeemURL.String())
	assert.Equal(t, "",
		p.Data().ValidateURL.String())
}

func TestNextcloudProviderOverrides(t *testing.T) {
	p := NewNextcloudProvider(
		&ProviderData{
			LoginURL: &url.URL{
				Scheme: "https",
				Host:   "example.com",
				Path:   "/index.php/apps/oauth2/authorize"},
			RedeemURL: &url.URL{
				Scheme: "https",
				Host:   "example.com",
				Path:   "/index.php/apps/oauth2/api/v1/token"},
			ValidateURL: &url.URL{
				Scheme:   "https",
				Host:     "example.com",
				Path:     "/test/ocs/v2.php/cloud/user",
				RawQuery: formatJSON},
			Scope: "profile"})
	assert.NotEqual(t, nil, p)
	assert.Equal(t, "Nextcloud", p.Data().ProviderName)
	assert.Equal(t, "https://example.com/index.php/apps/oauth2/authorize",
		p.Data().LoginURL.String())
	assert.Equal(t, "https://example.com/index.php/apps/oauth2/api/v1/token",
		p.Data().RedeemURL.String())
	assert.Equal(t, "https://example.com/test/ocs/v2.php/cloud/user?"+formatJSON,
		p.Data().ValidateURL.String())
}

func TestNextcloudProviderEnrichSession(t *testing.T) {
	b := testNextcloudBackend(`{ "ocs": { "data": {
    "id": "ausername",
    "email": "myemail@example.com",
    "groups": ["admin", "testgroup"],
    "displayname": "Test User"
  }}}`)
	defer b.Close()

	bURL, _ := url.Parse(b.URL)
	p := testNextcloudProvider(bURL.Host)
	p.ValidateURL.Path = userPath
	p.ValidateURL.RawQuery = formatJSON

	session := CreateAuthorizedSession()
	err := p.EnrichSession(context.Background(), session)
	assert.Equal(t, nil, err)
	assert.Equal(t, "myemail@example.com", session.Email)
	assert.Equal(t, "ausername", session.User)
	assert.Equal(t, []string{"admin", "testgroup"}, session.Groups)
}

// Note that trying to trigger the "failed building request" case is not
// practical, since the only way it can fail is if the URL fails to parse.
func TestNextcloudProviderEnrichSessionFailedRequest(t *testing.T) {
	b := testNextcloudBackend("unused payload")
	defer b.Close()

	bURL, _ := url.Parse(b.URL)
	p := testNextcloudProvider(bURL.Host)
	p.ValidateURL.Path = userPath
	p.ValidateURL.RawQuery = formatJSON

	// We'll trigger a request failure by using an unexpected access
	// token. Alternatively, we could allow the parsing of the payload as
	// JSON to fail.
	session := &sessions.SessionState{AccessToken: "unexpected_access_token"}
	err := p.EnrichSession(context.Background(), session)
	assert.NotEqual(t, nil, err)
	assert.Equal(t, "", session.Email)
	assert.Equal(t, "", session.User)
}

func TestNextcloudProviderEnrichSessionEmailNotPresentInPayload(t *testing.T) {
	b := testNextcloudBackend(`{"foo": "bar"}`)
	defer b.Close()

	bURL, _ := url.Parse(b.URL)
	p := testNextcloudProvider(bURL.Host)
	p.ValidateURL.Path = userPath
	p.ValidateURL.RawQuery = formatJSON

	session := CreateAuthorizedSession()
	err := p.EnrichSession(context.Background(), session)
	assert.NotEqual(t, nil, err)
	assert.Equal(t, "", session.Email)
	assert.Equal(t, "", session.User)
}
