package main

import (
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"time"

	"github.com/aws/aws-sdk-go/aws/endpoints"

	"github.com/aws/aws-sdk-go/aws/request"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/spf13/cobra"
)

// version to record build version
var version = "1.0.3"

// endpoint default Server URL
var endpoint = "http://s3test.myshare.io:9090"

// S3Client represent a Client
type S3Client struct {
	// credential file
	credential string
	// profile in credential file
	profile string
	// Server endpoine(URL)
	endpoint string
	// accessKey(username)
	accessKey string
	// secretKey(password)
	secretKey string
	// debug log
	debug bool
	// region
	region string
	useSSL bool
}

func (sc *S3Client) newS3Client() (*s3.S3, error) {
	var cred *credentials.Credentials
	if sc.accessKey != "" {
		cred = credentials.NewStaticCredentials(sc.accessKey, sc.secretKey, "")
	} else if sc.credential != "" {
		cred = credentials.NewSharedCredentials(sc.credential, sc.profile)
	} else if sc.profile != "" {
		cred = credentials.NewSharedCredentials("", sc.profile)
	}
	var logLevel *aws.LogLevelType
	if sc.debug {
		logLevel = aws.LogLevel(aws.LogDebug)
	}
	sess, err := session.NewSession(&aws.Config{
		Credentials:      cred,
		Endpoint:         aws.String(sc.endpoint),
		Region:           aws.String(sc.region),
		LogLevel:         logLevel,
		S3ForcePathStyle: aws.Bool(true),
	})
	if err != nil {
		log.Fatal("NewSession: ", err)
		return nil, err
	}
	return s3.New(sess), nil
}

func (sc *S3Client) createBucket(bucketName string) {
	svc, err := sc.newS3Client()
	if err != nil {
		log.Println("NewSession: ", err)
		return
	}
	cparams := &s3.CreateBucketInput{
		Bucket: aws.String(bucketName),
	}
	_, err = svc.CreateBucket(cparams)
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	fmt.Printf("Created bucket %s\n", bucketName)
}

func (sc *S3Client) headBucket(bucket string) {
	svc, err := sc.newS3Client()
	if err != nil {
		log.Println("NewSession: ", err)
		return
	}
	head, err := svc.HeadBucket(&s3.HeadBucketInput{
		Bucket: aws.String(bucket),
	})
	if err != nil {
		fmt.Printf("Failed to head Bucket %s, %s\n", bucket, err.Error())
		return
	}
	fmt.Println(head)
}

func (sc *S3Client) getBucketACL(bucket string) {
	svc, err := sc.newS3Client()
	if err != nil {
		log.Println("NewSession: ", err)
		return
	}
	acl, err := svc.GetBucketAcl(&s3.GetBucketAclInput{
		Bucket: aws.String(bucket),
	})
	if err != nil {
		fmt.Printf("Failed to get Bucket %s ACL, %s\n", bucket, err.Error())
		return
	}
	fmt.Println(acl)
}

func (sc *S3Client) listBucket() {
	svc, err := sc.newS3Client()
	if err != nil {
		log.Println("NewSession: ", err)
		return
	}

	bks, err := svc.ListBuckets(nil)
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	fmt.Printf("Bucket %v\n", *bks)
}

func (sc *S3Client) deleteBucket(bucket string) {
	if bucket == "" {
		log.Fatal("invalid bucket", bucket)
	}
	svc, err := sc.newS3Client()
	if err != nil {
		log.Fatal("init s3 client", err)
	}
	// Create Object
	_, err = svc.DeleteBucket(
		&s3.DeleteBucketInput{
			Bucket: aws.String(bucket),
		})
	if err != nil {
		fmt.Println(err)
	} else {
		fmt.Printf("bucket %s deleted\n", bucket)
	}
}

func (sc *S3Client) putObject(bucket, key, filename string, overwrite bool) {
	file, err := os.Open(filename)
	if err != nil {
		fmt.Println("Failed to open file", filename, err)
		os.Exit(1)
	}
	defer file.Close()
	if key == "" {
		key = filepath.Base(filename)
	}
	svc, err := sc.newS3Client()
	if err != nil {
		log.Println("NewSession: ", err)
		return
	}
	_, err = svc.PutObject(&s3.PutObjectInput{
		Body:   file,
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		fmt.Printf("Failed to upload Object %s/%s, %s\n", bucket, key, err.Error())
	} else {
		fmt.Printf("Uploaded Object %s\n", key)
	}
}

func (sc *S3Client) headObject(bucket, key string) {
	svc, err := sc.newS3Client()
	if err != nil {
		log.Println("NewSession: ", err)
		return
	}
	head, err := svc.HeadObject(&s3.HeadObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		fmt.Printf("Failed to head Object %s/%s, %s\n", bucket, key, err.Error())
		return
	}
	fmt.Println(head)
}

func (sc *S3Client) getObjectACL(bucket, key string) {
	svc, err := sc.newS3Client()
	if err != nil {
		log.Println("NewSession: ", err)
		return
	}
	acl, err := svc.GetObjectAcl(&s3.GetObjectAclInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		fmt.Printf("Failed to get Object %s/%s ACL, %s\n", bucket, key, err.Error())
		return
	}
	fmt.Println(acl)
}

func (sc *S3Client) mpuObject(bucket, key, filename string, overwrite bool) {
	file, err := os.Open(filename)
	if err != nil {
		fmt.Println("Failed to open file", filename, err)
		os.Exit(1)
	}
	defer file.Close()
	if key == "" {
		key = filepath.Base(filename)
	}

	svc, err := sc.newS3Client()
	if err != nil {
		log.Println("NewSession: ", err)
		return
	}

	uploader := s3manager.NewUploaderWithClient(svc)
	_, err = uploader.Upload(&s3manager.UploadInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
		Body:   file,
	})
	if err != nil {
		fmt.Printf("Failed to upload Object %s/%s, %s\n", bucket, key, err.Error())
		return
	}
	fmt.Printf("Uploaded Object %s\n", key)
}

func (sc *S3Client) listObject(bucket, prefix, delimiter string) {
	svc, err := sc.newS3Client()
	if err != nil {
		log.Println("NewSession: ", err)
		return
	}
	obj, err := svc.ListObjects(&s3.ListObjectsInput{
		Bucket:    aws.String(bucket),
		Prefix:    aws.String(prefix),
		Delimiter: aws.String(delimiter),
	})
	if err != nil {
		fmt.Println("Failed to list Object", err)
		return
	}
	fmt.Println(obj)
}

func (sc *S3Client) getObject(bucket, key, oRange, filename string) {
	if filename == "" {
		filename = key
	}
	file, err := os.Create(filename)
	if err != nil {
		log.Printf("Unable to open file %s, %v", filename, err)
		return
	}
	defer file.Close()

	svc, err := sc.newS3Client()
	if err != nil {
		log.Println("NewSession: ", err)
		return
	}
	input := &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	}
	if oRange != "" {
		input.SetRange(fmt.Sprintf("bytes=%s", oRange))
	}
	obj, err := svc.GetObject(input)
	if err != nil {
		fmt.Println("Failed to download Object", err)
		return
	}
	io.Copy(file, obj.Body)
	fmt.Printf("Download Object %s\n", key)
}

func (sc *S3Client) deleteObject(bucket, key string, prefix bool) (int64, error) {
	svc, err := sc.newS3Client()
	if err != nil {
		return 0, err
	}
	var cnt int64
	if prefix {
		for {
			objects := make([]*s3.ObjectIdentifier, 0, 1000)
			// use svc.ListObjectsPages() ?
			objs, err := svc.ListObjects(&s3.ListObjectsInput{
				Bucket: aws.String(bucket),
				Prefix: aws.String(key),
			})
			if err != nil {
				return cnt, err
			}
			objCnt := len(objs.Contents)
			if objCnt == 0 {
				return cnt, nil
			}
			for _, obj := range objs.Contents {
				objects = append(objects, &s3.ObjectIdentifier{Key: obj.Key})
			}
			_, err = svc.DeleteObjects(&s3.DeleteObjectsInput{
				Bucket: aws.String(bucket),
				Delete: &s3.Delete{Objects: objects, Quiet: aws.Bool(true)},
			})
			if err != nil {
				return cnt, err
			}
			cnt = cnt + int64(objCnt)
		}
	} else {
		_, err = svc.DeleteObject(&s3.DeleteObjectInput{
			Bucket: aws.String(bucket),
			Key:    aws.String(key),
		})
	}
	return 0, err
}

func (sc *S3Client) aclObject(bucket, key string, prefix bool) (int64, error) {
	svc, err := sc.newS3Client()
	if err != nil {
		return 0, err
	}
	var cnt int64
	if prefix {
		for {
			objects := make([]*s3.ObjectIdentifier, 0, 1000)
			objs, err := svc.ListObjects(&s3.ListObjectsInput{
				Bucket: aws.String(bucket),
				Prefix: aws.String(key),
			})
			if err != nil {
				return cnt, err
			}
			objCnt := len(objs.Contents)
			if objCnt == 0 {
				return cnt, nil
			}
			for _, obj := range objs.Contents {
				objects = append(objects, &s3.ObjectIdentifier{Key: obj.Key})
			}
			_, err = svc.DeleteObjects(&s3.DeleteObjectsInput{
				Bucket: aws.String(bucket),
				Delete: &s3.Delete{Objects: objects, Quiet: aws.Bool(true)},
			})
			if err != nil {
				return cnt, err
			}
			cnt = cnt + int64(objCnt)
		}
	} else {
		_, err = svc.DeleteObject(&s3.DeleteObjectInput{
			Bucket: aws.String(bucket),
			Key:    aws.String(key),
		})
	}
	return 0, err
}

func (sc *S3Client) presignObject(bucket, key string, exp time.Duration, put bool) (string, error) {
	svc, err := sc.newS3Client()
	if err != nil {
		return "", err
	}
	var req *request.Request
	if put {
		// presign a PUT URL to upload Object
		req, _ = svc.PutObjectRequest(&s3.PutObjectInput{
			Bucket: aws.String(bucket),
			Key:    aws.String(key),
		})
	} else {
		req, _ = svc.GetObjectRequest(&s3.GetObjectInput{
			Bucket: aws.String(bucket),
			Key:    aws.String(key),
		})
	}
	return req.Presign(exp)
}

func (sc *S3Client) presignObjectV2(bucket, key string, exp time.Duration, put bool) (string, error) {
	if sc.accessKey == "" {
		return "", errors.New("unknow access key")
	} else if sc.secretKey == "" {
		return "", errors.New("unknow secret key")
	} else if sc.endpoint == "" {
		return "", errors.New("unknow endpoint")
	}
	u, err := url.Parse(fmt.Sprintf("%s/%s/%s", sc.endpoint, bucket, key))
	if err != nil {
		return "", err
	}
	method := http.MethodGet
	if put {
		method = http.MethodPut
	}
	return Presign2(u, method, sc.accessKey, sc.secretKey, exp)
}

func main() {
	sc := S3Client{}
	var rootCmd = &cobra.Command{
		Use:     "s3cli",
		Short:   "s3cli client tool",
		Long:    "s3cli client tool for S3 Bucket/Object operation",
		Version: fmt.Sprintf("[%s]", version),
	}
	rootCmd.PersistentFlags().BoolVarP(&sc.debug, "debug", "d", false, "print debug log")
	rootCmd.PersistentFlags().StringVarP(&sc.credential, "credential", "c", "", "credentail file")
	rootCmd.PersistentFlags().StringVarP(&sc.profile, "profile", "p", "", "credentail profile")
	rootCmd.PersistentFlags().StringVarP(&sc.endpoint, "endpoint", "e", endpoint, "endpoint")
	rootCmd.PersistentFlags().StringVarP(&sc.accessKey, "accesskey", "a", "", "access key")
	rootCmd.PersistentFlags().StringVarP(&sc.secretKey, "secretkey", "s", "", "secret key")
	rootCmd.PersistentFlags().StringVarP(&sc.region, "region", "R", endpoints.CnNorth1RegionID, "s3 region")
	rootCmd.Flags().BoolP("version", "v", false, "print version")

	createBucketCmd := &cobra.Command{
		Use:     "createBucket <name>",
		Aliases: []string{"cb"},
		Short:   "create Bucket",
		Long:    "create Bucket",
		Args:    cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			sc.createBucket(args[0])
		},
	}
	rootCmd.AddCommand(createBucketCmd)

	listBucketCmd := &cobra.Command{
		Use:     "listBucket",
		Aliases: []string{"lb"},
		Short:   "list Buckets",
		Long:    "list all Buckets",
		Args:    cobra.ExactArgs(0),
		Run: func(cmd *cobra.Command, args []string) {
			sc.listBucket()
		},
	}
	rootCmd.AddCommand(listBucketCmd)

	deleteBucketCmd := &cobra.Command{
		Use:     "deleteBucket <bucket>",
		Aliases: []string{"db"},
		Short:   "delete bucket",
		Long:    "delete a bucket",
		Args:    cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			sc.deleteBucket(args[0])
		},
	}
	rootCmd.AddCommand(deleteBucketCmd)

	headCmd := &cobra.Command{
		Use:     "head <bucket> [key]",
		Aliases: []string{"head"},
		Short:   "head Bucket/Object",
		Long:    "get Bucket/Object metadata",
		Args:    cobra.RangeArgs(1, 2),
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) == 2 {
				sc.headObject(args[0], args[1])
			} else {
				sc.headBucket(args[0])
			}
		},
	}
	rootCmd.AddCommand(headCmd)

	getaclCmd := &cobra.Command{
		Use:     "getacl <bucket> [key]",
		Aliases: []string{"ga"},
		Short:   "get Bucket/Object acl",
		Long:    "get Bucket/Object ACL",
		Args:    cobra.RangeArgs(1, 2),
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) == 2 {
				sc.getObjectACL(args[0], args[1])
			} else {
				sc.getBucketACL(args[0])
			}
		},
	}
	rootCmd.AddCommand(getaclCmd)

	putObjectCmd := &cobra.Command{
		Use:     "upload <bucket> <local-file>",
		Aliases: []string{"up"},
		Short:   "upload Object",
		Long:    "upload Object to Bucket",
		Args:    cobra.ExactArgs(2),
		Run: func(cmd *cobra.Command, args []string) {
			key := cmd.Flag("key").Value.String()
			sc.putObject(args[0], key, args[1], cmd.Flag("overwrite").Changed)
		},
	}
	putObjectCmd.Flags().StringP("key", "k", "", "key name")
	putObjectCmd.Flags().BoolP("overwrite", "w", false, "overwrite file if exist")
	rootCmd.AddCommand(putObjectCmd)

	mpuObjectCmd := &cobra.Command{
		Use:     "mpu <bucket> <local-file>",
		Aliases: []string{"mp", "mu"},
		Short:   "mpu Object",
		Long:    "mutiPartUpload Object to Bucket",
		Args:    cobra.ExactArgs(2),
		Run: func(cmd *cobra.Command, args []string) {
			key := cmd.Flag("key").Value.String()
			sc.mpuObject(args[0], key, args[1], cmd.Flag("overwrite").Changed)
		},
	}
	mpuObjectCmd.Flags().StringP("key", "k", "", "key name")
	mpuObjectCmd.Flags().BoolP("overwrite", "w", false, "overwrite file if exist")
	rootCmd.AddCommand(mpuObjectCmd)

	listObjectCmd := &cobra.Command{
		Use:     "list [bucket]",
		Aliases: []string{"ls"},
		Short:   "list Buckets or Objects in Bucket",
		Long:    "list Buckets or Objects in Bucket",
		Args:    cobra.RangeArgs(0, 1),
		Run: func(cmd *cobra.Command, args []string) {
			prefix := cmd.Flag("prefix").Value.String()
			delimiter := cmd.Flag("delimiter").Value.String()
			if len(args) == 1 {
				sc.listObject(args[0], prefix, delimiter)
			} else {
				sc.listBucket()
			}
		},
	}
	listObjectCmd.Flags().StringP("prefix", "P", "", "Object prefix")
	listObjectCmd.Flags().StringP("delimiter", "", "", "Object delimiter")
	rootCmd.AddCommand(listObjectCmd)

	getObjectCmd := &cobra.Command{
		Use:     "download <bucket> <key> [destination]",
		Aliases: []string{"get", "down", "d"},
		Short:   "download Object",
		Long:    "downlaod Object from Bucket",
		Args:    cobra.RangeArgs(2, 3),
		Run: func(cmd *cobra.Command, args []string) {
			destination := ""
			if len(args) == 3 {
				destination = args[2]
			}
			objRange := cmd.Flag("range").Value.String()
			sc.getObject(args[0], args[1], objRange, destination)
		},
	}
	getObjectCmd.Flags().StringP("range", "r", "", "Object range to download, 0-64 means [0, 64]")
	getObjectCmd.Flags().BoolP("overwrite", "w", false, "overwrite file if exist")
	rootCmd.AddCommand(getObjectCmd)

	deleteObjectCmd := &cobra.Command{
		Use:     "delete <bucket> [key|prefix]",
		Aliases: []string{"del", "rm"},
		Short:   "delete Bucket or Object",
		Long:    "delete Bucket or Object(s) in Bucket",
		Args:    cobra.RangeArgs(1, 2),
		Run: func(cmd *cobra.Command, args []string) {
			prefix := cmd.Flag("prefix").Changed
			if len(args) == 2 {
				if cnt, err := sc.deleteObject(args[0], args[1], prefix); err != nil {
					fmt.Println("delete Object error: ", err)
				} else {
					fmt.Printf("delete %d Objects success\n", cnt)
				}
			} else if prefix {
				if cnt, err := sc.deleteObject(args[0], "", prefix); err != nil {
					fmt.Println("delete Object error: ", err)
				} else {
					fmt.Printf("delete %d Objects success\n", cnt)
				}
			} else {
				sc.deleteBucket(args[0])
			}
		},
	}
	deleteObjectCmd.Flags().BoolP("prefix", "P", false, "delete all Objects with specified prefix(key)")
	rootCmd.AddCommand(deleteObjectCmd)

	presignObjectCmd := &cobra.Command{
		Use:     "presign <bucket> <key>",
		Aliases: []string{"psn", "psg"},
		Short:   "presign Object",
		Long:    "presign Object URL",
		Args:    cobra.ExactArgs(2),
		Run: func(cmd *cobra.Command, args []string) {
			exp, err := time.ParseDuration(cmd.Flag("expire").Value.String())
			if err != nil {
				fmt.Println("invalid expire : ", err)
				return
			}
			url := ""
			if cmd.Flag("v2").Changed {
				url, err = sc.presignObjectV2(args[0], args[1], exp, cmd.Flag("put").Changed)
			} else {
				url, err = sc.presignObject(args[0], args[1], exp, cmd.Flag("put").Changed)
			}
			if err != nil {
				fmt.Println("presign failed: ", err)
			} else {
				fmt.Println(url)
			}
		},
	}
	presignObjectCmd.Flags().DurationP("expire", "E", 12*time.Hour, "URL expire time")
	presignObjectCmd.Flags().BoolP("put", "", false, "generate a put URL")
	presignObjectCmd.Flags().BoolP("v2", "2", false, "s3v2 signature")
	rootCmd.AddCommand(presignObjectCmd)

	aclObjectCmd := &cobra.Command{
		Use:     "acl <bucket> [key|prefix]",
		Aliases: []string{"pa"},
		Short:   "acl Bucket or Object",
		Long:    "acl Bucket or Object(s) in Bucket",
		Args:    cobra.RangeArgs(1, 2),
		Run: func(cmd *cobra.Command, args []string) {
			prefix := cmd.Flag("prefix").Changed
			key := ""
			if len(args) == 2 {
				key = args[1]
			}
			if cnt, err := sc.aclObject(args[0], key, prefix); err != nil {
				fmt.Println("acl Object error: ", err)
			} else {
				fmt.Printf("acl %d Objects success\n", cnt)
			}

		},
	}
	aclObjectCmd.Flags().BoolP("prefix", "P", false, "acl all Objects with specified prefix(key)")
	rootCmd.AddCommand(aclObjectCmd)

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
