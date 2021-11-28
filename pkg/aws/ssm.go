package aws

import (
	"log"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ssm"
)

type Parameter struct {
	Name  *string `min:"1" type:"string"`
	Value *string `type:"string" sensityve:"true"`
}

// GetParameter - Retrive the value for a given SSM Parameter name
func GetParameter(parameterName string) *string {
	ssmService := ssm.New(sharedSession)
	getParameterOutput, err := ssmService.GetParameter(&ssm.GetParameterInput{Name: &parameterName})
	if err != nil {
		log.Println(err)
		return aws.String("")
	}
	return getParameterOutput.Parameter.Value
}

// GetParametersByPath - Get parameters value by SSM Parameter path
func GetParametersByPath(parameterPath string) []Parameter {
	ssmService := ssm.New(sharedSession)
	parameters := make([]Parameter, 0)
	getParametersByPathInputReq := ssm.GetParametersByPathInput{
		Path: &parameterPath,
	}
	for {
		getParametersByPathInputResp, err := ssmService.GetParametersByPath(&getParametersByPathInputReq)
		if err != nil {
			log.Fatalf("Failed to get parameters for %s: %v\n", parameterPath, err)
			return nil
		}
		for _, s := range getParametersByPathInputResp.Parameters {
			parameters = append(parameters, Parameter{
				Name:  s.Name,
				Value: s.Value,
			})
		}
		getParametersByPathInputReq.NextToken = getParametersByPathInputResp.NextToken
		if aws.StringValue(getParametersByPathInputReq.NextToken) == "" {
			break
		}
	}
	return parameters
}
