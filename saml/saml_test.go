package saml

import (
	"encoding/base64"
	"testing"
	"time"
)

func TestXMLUnmarshal(t *testing.T) {
	xml := `<saml2p:Response>
              <saml2:Assertion>
                <saml2:Conditions NotBefore="2018-04-04T23:49:54.598Z" NotOnOrAfter="2018-04-04T23:59:54.598Z">
                </saml2:Conditions>
                <saml2:AttributeStatement>
                  <saml2:Attribute Name="camelid">
                    <saml2:AttributeValue>llama</saml2:AttributeValue>
                  </saml2:Attribute>
                  <saml2:Attribute Name="cheeses">
                    <saml2:AttributeValue>manchego</saml2:AttributeValue>
                    <saml2:AttributeValue>reypenaer</saml2:AttributeValue>
                  </saml2:Attribute>
                </saml2:AttributeStatement>
              </saml2:Assertion>
            </saml2p:Response>`

	response, err := ParseResponse(xml)

	if err != nil {
		t.Log("---------------")
		t.Log("Failed to parse SAML at all!")
		t.Errorf("Got error: %s", err)
	}

	conditions := response.Assertion.Conditions

	expectedNotBefore, _ := time.Parse(time.RFC3339, "2018-04-04T23:49:54.598Z")

	if conditions.NotBefore != expectedNotBefore {
		t.Log("---------------")
		t.Log("Failed to parse the NotBefore condition")
		t.Logf("Expected: %s", expectedNotBefore)
		t.Logf("Got: %s", conditions.NotBefore)
		t.Fail()
	}

	expectedNotOnOrAfter, _ := time.Parse(time.RFC3339, "2018-04-04T23:59:54.598Z")

	if conditions.NotOnOrAfter != expectedNotOnOrAfter {
		t.Log("---------------")
		t.Log("Failed to parse the NotOnOrAfter condition")
		t.Logf("Expected: %s", expectedNotOnOrAfter)
		t.Logf("Got: %s", conditions.NotOnOrAfter)
		t.Fail()
	}

	attributes := response.Assertion.Attributes

	expectedAttributes := []samlAssertionAttribute{
		samlAssertionAttribute{"camelid", []string{"llama"}},
		samlAssertionAttribute{"cheeses", []string{"manchego", "reypenaer"}},
	}

	if len(attributes) != len(expectedAttributes) {
		t.Log("---------------")
		t.Log("Parsed the wrong number of attributes from the XML!")
		t.Logf("Expected: %d", len(expectedAttributes))
		t.Logf("Got: %d", len(attributes))
		t.Fail()
	}

	for index, attribute := range attributes {
		expectedAttribute := expectedAttributes[index]

		if attribute.Name != expectedAttribute.Name {
			t.Log("---------------")
			t.Logf("Did not correctly parse out name of attribute #%d", index)
			t.Logf("Expected: %s", expectedAttribute.Name)
			t.Logf("Got: %s", attribute.Name)
			t.Fail()
		}

		if len(attribute.Values) != len(expectedAttribute.Values) {
			t.Log("---------------")
			t.Logf("Did not correctly parse out the values of attribute #%d", index)
			t.Logf("Expected: %s", expectedAttribute.Values)
			t.Logf("Got: %s", attribute.Values)
			t.Fail()
		}

		for vIndex, value := range attribute.Values {
			expectedValue := expectedAttribute.Values[vIndex]

			if value != expectedValue {
				t.Log("---------------")
				t.Logf("Did not correctly parse out the values of attribute #%d", index)
				t.Logf("Expected: %s", attribute.Values)
				t.Logf("Got: %s", expectedAttribute.Values)
				t.Fail()
			}
		}
	}
}

func TestCreateLoginData(t *testing.T) {
	payload := "abloboftext"
	encodedPayload := base64.StdEncoding.EncodeToString([]byte(payload))
	roleAttribute := samlAssertionAttribute{
		Name: "https://aws.amazon.com/SAML/Attributes/Role",
		Values: []string{
			"cheese,manchego",
			"cheese,reypenaer",
		},
	}

	response := samlResponse{
		Assertion: samlAssertion{
			Conditions: samlAssertionConditions{},
			Attributes: []samlAssertionAttribute{
				roleAttribute,
			},
		},
	}

	subject := CreateLoginData(response, payload)

	if subject.Assertion != encodedPayload {
		t.Log("---------------")
		t.Log("Did not correctly encode the SAML assertion")
		t.Logf("Expected: %s", encodedPayload)
		t.Logf("Got: %s", subject.Assertion)
		t.Fail()
	}

	if len(subject.Roles) != len(roleAttribute.Values) {
		t.Log("---------------")
		t.Log("Did not correctly translate the list of roles ")
		t.Logf("Expected length: %d", len(roleAttribute.Values))
		t.Logf("Got: %s", subject.Roles)
		t.Fail()
	}

	for index, value := range roleAttribute.Values {
		role := subject.Roles[index]
		actualRoleString := role.PrincipalArn + "," + role.RoleArn

		if actualRoleString != value {
			t.Log("---------------")
			t.Log("A role was not correctly parsed out of the SAML response")
			t.Logf("Expected: %s", value)
			t.Logf("Got: %s", actualRoleString)
			t.Fail()
		}
	}
}

func TestGetLoginRole(t *testing.T) {
	expectedRole := LoginRole{
		RoleArn:      "aws:arn:llama",
		PrincipalArn: "aws:arn:principal",
	}

	subject := LoginData{
		Roles: []LoginRole{
			expectedRole,
		},
	}

	actualRole, err := subject.GetLoginRole("aws:arn:llama")

	if err != nil {
		t.Log("---------------")
		t.Log("GetLoginRole didn't return the correct role!")
		t.Logf("Expected: %s", expectedRole.RoleArn)
		t.Log("Got nothing!")
		t.Fail()
	} else if actualRole != expectedRole {
		t.Log("---------------")
		t.Log("GetLoginRole didn't return the correct role!")
		t.Logf("Expected: %s", expectedRole.RoleArn)
		t.Logf("Got: %s", actualRole.RoleArn)
		t.Fail()
	}

	actualRole, err = subject.GetLoginRole("aws:arn:notarole")

	if err == nil {
		t.Log("---------------")
		t.Log("GetLoginRole returned a role when it shouldn't have!")
		t.Log("Expected nothing")
		t.Logf("Got: %s", actualRole.RoleArn)
		t.Fail()
	}
}
