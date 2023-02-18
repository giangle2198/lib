package testing

import (
	"log"
	"net/http"
	"testing"
)

// TableTestDownload validates the http Get function can download content
func TestTableDownload(t *testing.T) {
	var urls = []struct {
		url        string
		statusCode int
	}{
		{
			"http://www.goinggo.net/feeds/posts/default?alt=rss",
			http.StatusOK,
		},
		{
			"http://rss.cnn.com/rss/cnn_topstbadurl.rss",
			http.StatusNotFound,
		},
	}

	t.Log("Given the need to test downloading content")
	{
		for _, u := range urls {

			t.Logf("\tWhen checking \"%s\" for status code \"%d\"", u.url, u.statusCode)
			{

				resp, err := http.Get(u.url)
				if err != nil {
					log.Fatal("\t\tShould be able to make the Get call.", ballotX, err)
				}
				t.Log("\t\tShould be able to make the Get call.", checkMark)

				defer resp.Body.Close()

				if resp.StatusCode == u.statusCode {
					t.Logf("\t\tShould receive a \"%d\" status. %v", u.statusCode, checkMark)
				} else {
					t.Errorf("\t\tShould receive a \"%d\" status. %v %v", u.statusCode, ballotX, resp.StatusCode)
				}
			}

		}
	}

}
