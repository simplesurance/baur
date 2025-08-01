// Code generated by smithy-go-codegen DO NOT EDIT.

package s3

import (
	"context"
	"fmt"
	awsmiddleware "github.com/aws/aws-sdk-go-v2/aws/middleware"
	"github.com/aws/aws-sdk-go-v2/aws/signer/v4"
	s3cust "github.com/aws/aws-sdk-go-v2/service/s3/internal/customizations"
	"github.com/aws/smithy-go/middleware"
	"github.com/aws/smithy-go/ptr"
	smithyhttp "github.com/aws/smithy-go/transport/http"
)

// Deletes the policy of a specified bucket.
//
// Directory buckets - For directory buckets, you must make requests for this API
// operation to the Regional endpoint. These endpoints support path-style requests
// in the format https://s3express-control.region-code.amazonaws.com/bucket-name .
// Virtual-hosted-style requests aren't supported. For more information about
// endpoints in Availability Zones, see [Regional and Zonal endpoints for directory buckets in Availability Zones]in the Amazon S3 User Guide. For more
// information about endpoints in Local Zones, see [Concepts for directory buckets in Local Zones]in the Amazon S3 User Guide.
//
// Permissions If you are using an identity other than the root user of the Amazon
// Web Services account that owns the bucket, the calling identity must both have
// the DeleteBucketPolicy permissions on the specified bucket and belong to the
// bucket owner's account in order to use this operation.
//
// If you don't have DeleteBucketPolicy permissions, Amazon S3 returns a 403
// Access Denied error. If you have the correct permissions, but you're not using
// an identity that belongs to the bucket owner's account, Amazon S3 returns a 405
// Method Not Allowed error.
//
// To ensure that bucket owners don't inadvertently lock themselves out of their
// own buckets, the root principal in a bucket owner's Amazon Web Services account
// can perform the GetBucketPolicy , PutBucketPolicy , and DeleteBucketPolicy API
// actions, even if their bucket policy explicitly denies the root principal's
// access. Bucket owner root principals can only be blocked from performing these
// API actions by VPC endpoint policies and Amazon Web Services Organizations
// policies.
//
//   - General purpose bucket permissions - The s3:DeleteBucketPolicy permission is
//     required in a policy. For more information about general purpose buckets bucket
//     policies, see [Using Bucket Policies and User Policies]in the Amazon S3 User Guide.
//
//   - Directory bucket permissions - To grant access to this API operation, you
//     must have the s3express:DeleteBucketPolicy permission in an IAM identity-based
//     policy instead of a bucket policy. Cross-account access to this API operation
//     isn't supported. This operation can only be performed by the Amazon Web Services
//     account that owns the resource. For more information about directory bucket
//     policies and permissions, see [Amazon Web Services Identity and Access Management (IAM) for S3 Express One Zone]in the Amazon S3 User Guide.
//
// HTTP Host header syntax  Directory buckets - The HTTP Host header syntax is
// s3express-control.region-code.amazonaws.com .
//
// # The following operations are related to DeleteBucketPolicy
//
// [CreateBucket]
//
// [DeleteObject]
//
// [Concepts for directory buckets in Local Zones]: https://docs.aws.amazon.com/AmazonS3/latest/userguide/s3-lzs-for-directory-buckets.html
// [DeleteObject]: https://docs.aws.amazon.com/AmazonS3/latest/API/API_DeleteObject.html
// [Using Bucket Policies and User Policies]: https://docs.aws.amazon.com/AmazonS3/latest/dev/using-iam-policies.html
// [CreateBucket]: https://docs.aws.amazon.com/AmazonS3/latest/API/API_CreateBucket.html
// [Regional and Zonal endpoints for directory buckets in Availability Zones]: https://docs.aws.amazon.com/AmazonS3/latest/userguide/endpoint-directory-buckets-AZ.html
// [Amazon Web Services Identity and Access Management (IAM) for S3 Express One Zone]: https://docs.aws.amazon.com/AmazonS3/latest/userguide/s3-express-security-iam.html
func (c *Client) DeleteBucketPolicy(ctx context.Context, params *DeleteBucketPolicyInput, optFns ...func(*Options)) (*DeleteBucketPolicyOutput, error) {
	if params == nil {
		params = &DeleteBucketPolicyInput{}
	}

	result, metadata, err := c.invokeOperation(ctx, "DeleteBucketPolicy", params, optFns, c.addOperationDeleteBucketPolicyMiddlewares)
	if err != nil {
		return nil, err
	}

	out := result.(*DeleteBucketPolicyOutput)
	out.ResultMetadata = metadata
	return out, nil
}

type DeleteBucketPolicyInput struct {

	// The bucket name.
	//
	// Directory buckets - When you use this operation with a directory bucket, you
	// must use path-style requests in the format
	// https://s3express-control.region-code.amazonaws.com/bucket-name .
	// Virtual-hosted-style requests aren't supported. Directory bucket names must be
	// unique in the chosen Zone (Availability Zone or Local Zone). Bucket names must
	// also follow the format bucket-base-name--zone-id--x-s3 (for example,
	// DOC-EXAMPLE-BUCKET--usw2-az1--x-s3 ). For information about bucket naming
	// restrictions, see [Directory bucket naming rules]in the Amazon S3 User Guide
	//
	// [Directory bucket naming rules]: https://docs.aws.amazon.com/AmazonS3/latest/userguide/directory-bucket-naming-rules.html
	//
	// This member is required.
	Bucket *string

	// The account ID of the expected bucket owner. If the account ID that you provide
	// does not match the actual owner of the bucket, the request fails with the HTTP
	// status code 403 Forbidden (access denied).
	//
	// For directory buckets, this header is not supported in this API operation. If
	// you specify this header, the request fails with the HTTP status code 501 Not
	// Implemented .
	ExpectedBucketOwner *string

	noSmithyDocumentSerde
}

func (in *DeleteBucketPolicyInput) bindEndpointParams(p *EndpointParameters) {

	p.Bucket = in.Bucket
	p.UseS3ExpressControlEndpoint = ptr.Bool(true)
}

type DeleteBucketPolicyOutput struct {
	// Metadata pertaining to the operation's result.
	ResultMetadata middleware.Metadata

	noSmithyDocumentSerde
}

func (c *Client) addOperationDeleteBucketPolicyMiddlewares(stack *middleware.Stack, options Options) (err error) {
	if err := stack.Serialize.Add(&setOperationInputMiddleware{}, middleware.After); err != nil {
		return err
	}
	err = stack.Serialize.Add(&awsRestxml_serializeOpDeleteBucketPolicy{}, middleware.After)
	if err != nil {
		return err
	}
	err = stack.Deserialize.Add(&awsRestxml_deserializeOpDeleteBucketPolicy{}, middleware.After)
	if err != nil {
		return err
	}
	if err := addProtocolFinalizerMiddlewares(stack, options, "DeleteBucketPolicy"); err != nil {
		return fmt.Errorf("add protocol finalizers: %v", err)
	}

	if err = addlegacyEndpointContextSetter(stack, options); err != nil {
		return err
	}
	if err = addSetLoggerMiddleware(stack, options); err != nil {
		return err
	}
	if err = addClientRequestID(stack); err != nil {
		return err
	}
	if err = addComputeContentLength(stack); err != nil {
		return err
	}
	if err = addResolveEndpointMiddleware(stack, options); err != nil {
		return err
	}
	if err = addComputePayloadSHA256(stack); err != nil {
		return err
	}
	if err = addRetry(stack, options); err != nil {
		return err
	}
	if err = addRawResponseToMetadata(stack); err != nil {
		return err
	}
	if err = addRecordResponseTiming(stack); err != nil {
		return err
	}
	if err = addSpanRetryLoop(stack, options); err != nil {
		return err
	}
	if err = addClientUserAgent(stack, options); err != nil {
		return err
	}
	if err = smithyhttp.AddErrorCloseResponseBodyMiddleware(stack); err != nil {
		return err
	}
	if err = smithyhttp.AddCloseResponseBodyMiddleware(stack); err != nil {
		return err
	}
	if err = addSetLegacyContextSigningOptionsMiddleware(stack); err != nil {
		return err
	}
	if err = addPutBucketContextMiddleware(stack); err != nil {
		return err
	}
	if err = addTimeOffsetBuild(stack, c); err != nil {
		return err
	}
	if err = addUserAgentRetryMode(stack, options); err != nil {
		return err
	}
	if err = addIsExpressUserAgent(stack); err != nil {
		return err
	}
	if err = addCredentialSource(stack, options); err != nil {
		return err
	}
	if err = addOpDeleteBucketPolicyValidationMiddleware(stack); err != nil {
		return err
	}
	if err = stack.Initialize.Add(newServiceMetadataMiddleware_opDeleteBucketPolicy(options.Region), middleware.Before); err != nil {
		return err
	}
	if err = addMetadataRetrieverMiddleware(stack); err != nil {
		return err
	}
	if err = addRecursionDetection(stack); err != nil {
		return err
	}
	if err = addDeleteBucketPolicyUpdateEndpoint(stack, options); err != nil {
		return err
	}
	if err = addResponseErrorMiddleware(stack); err != nil {
		return err
	}
	if err = v4.AddContentSHA256HeaderMiddleware(stack); err != nil {
		return err
	}
	if err = disableAcceptEncodingGzip(stack); err != nil {
		return err
	}
	if err = addRequestResponseLogging(stack, options); err != nil {
		return err
	}
	if err = addDisableHTTPSMiddleware(stack, options); err != nil {
		return err
	}
	if err = addSerializeImmutableHostnameBucketMiddleware(stack, options); err != nil {
		return err
	}
	if err = addInterceptBeforeRetryLoop(stack, options); err != nil {
		return err
	}
	if err = addInterceptAttempt(stack, options); err != nil {
		return err
	}
	if err = addInterceptExecution(stack, options); err != nil {
		return err
	}
	if err = addInterceptBeforeSerialization(stack, options); err != nil {
		return err
	}
	if err = addInterceptAfterSerialization(stack, options); err != nil {
		return err
	}
	if err = addInterceptBeforeSigning(stack, options); err != nil {
		return err
	}
	if err = addInterceptAfterSigning(stack, options); err != nil {
		return err
	}
	if err = addInterceptTransmit(stack, options); err != nil {
		return err
	}
	if err = addInterceptBeforeDeserialization(stack, options); err != nil {
		return err
	}
	if err = addInterceptAfterDeserialization(stack, options); err != nil {
		return err
	}
	if err = addSpanInitializeStart(stack); err != nil {
		return err
	}
	if err = addSpanInitializeEnd(stack); err != nil {
		return err
	}
	if err = addSpanBuildRequestStart(stack); err != nil {
		return err
	}
	if err = addSpanBuildRequestEnd(stack); err != nil {
		return err
	}
	return nil
}

func (v *DeleteBucketPolicyInput) bucket() (string, bool) {
	if v.Bucket == nil {
		return "", false
	}
	return *v.Bucket, true
}

func newServiceMetadataMiddleware_opDeleteBucketPolicy(region string) *awsmiddleware.RegisterServiceMetadata {
	return &awsmiddleware.RegisterServiceMetadata{
		Region:        region,
		ServiceID:     ServiceID,
		OperationName: "DeleteBucketPolicy",
	}
}

// getDeleteBucketPolicyBucketMember returns a pointer to string denoting a
// provided bucket member valueand a boolean indicating if the input has a modeled
// bucket name,
func getDeleteBucketPolicyBucketMember(input interface{}) (*string, bool) {
	in := input.(*DeleteBucketPolicyInput)
	if in.Bucket == nil {
		return nil, false
	}
	return in.Bucket, true
}
func addDeleteBucketPolicyUpdateEndpoint(stack *middleware.Stack, options Options) error {
	return s3cust.UpdateEndpoint(stack, s3cust.UpdateEndpointOptions{
		Accessor: s3cust.UpdateEndpointParameterAccessor{
			GetBucketFromInput: getDeleteBucketPolicyBucketMember,
		},
		UsePathStyle:                   options.UsePathStyle,
		UseAccelerate:                  options.UseAccelerate,
		SupportsAccelerate:             true,
		TargetS3ObjectLambda:           false,
		EndpointResolver:               options.EndpointResolver,
		EndpointResolverOptions:        options.EndpointOptions,
		UseARNRegion:                   options.UseARNRegion,
		DisableMultiRegionAccessPoints: options.DisableMultiRegionAccessPoints,
	})
}
