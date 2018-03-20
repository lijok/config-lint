package main

import (
	"encoding/json"
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/lhitchon/config-lint/assertion"
)

type SecurityGroupLinter struct {
	Log assertion.LoggingFunction
}

func loadSecurityGroupResources(log assertion.LoggingFunction) []assertion.Resource {
	region := &aws.Config{Region: aws.String("us-east-1")}
	awsSession := session.New()
	ec2Client := ec2.New(awsSession, region)
	response, err := ec2Client.DescribeSecurityGroups(&ec2.DescribeSecurityGroupsInput{})
	if err != nil {
		panic(err)
	}
	resources := make([]assertion.Resource, 0)
	for _, securityGroup := range response.SecurityGroups {

		// convert to JSON string
		jsonData, err := json.Marshal(securityGroup)
		if err != nil {
			panic(err)
		}

		// then convert to an interface{}
		// seem to need this for JMESPath to work properly
		var data interface{}
		err = json.Unmarshal(jsonData, &data)
		if err != nil {
			panic(err)
		}

		r := assertion.Resource{
			Id:         *securityGroup.GroupId,
			Type:       "AWS::EC2::SecurityGroup",
			Properties: data,
		}
		resources = append(resources, r)
	}
	return resources
}

func (l SecurityGroupLinter) ValidateSecurityGroupResources(resources []assertion.Resource, rules []assertion.Rule, tags []string) []assertion.Violation {

	valueSource := assertion.StandardValueSource{Log: l.Log}
	filteredRules := assertion.FilterRulesByTag(rules, tags)
	resolvedRules := assertion.ResolveRules(filteredRules, valueSource, l.Log)

	allViolations := make([]assertion.Violation, 0)
	for _, rule := range resolvedRules {
		l.Log(fmt.Sprintf("Rule %s: %s", rule.Id, rule.Message))
		for _, resource := range assertion.FilterResourcesByType(resources, rule.Resource) {
			if assertion.ExcludeResource(rule, resource) {
				l.Log(fmt.Sprintf("Ignoring resource %s", resource.Id))
			} else {
				_, violations := assertion.CheckRule(rule, resource, l.Log)
				allViolations = append(allViolations, violations...)
			}
		}
	}
	return allViolations
}

func (l SecurityGroupLinter) Validate(filenames []string, ruleSet assertion.RuleSet, tags []string, ruleIds []string) ([]string, []assertion.Violation) {
	rules := assertion.FilterRulesById(ruleSet.Rules, ruleIds)
	resources := loadSecurityGroupResources(l.Log)
	violations := l.ValidateSecurityGroupResources(resources, rules, tags)
	return []string{}, violations
}

func (l SecurityGroupLinter) Search(filenames []string, ruleSet assertion.RuleSet, searchExpression string) {
	resources := loadSecurityGroupResources(l.Log)
	for _, resource := range resources {
		v, err := assertion.SearchData(searchExpression, resource.Properties)
		if err != nil {
			fmt.Println(err)
		} else {
			s, err := assertion.JSONStringify(v)
			if err != nil {
				fmt.Println(err)
			} else {
				fmt.Printf("%s: %s\n", resource.Id, s)
			}
		}
	}
}