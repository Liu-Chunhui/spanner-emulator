package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"

	database "cloud.google.com/go/spanner/admin/database/apiv1"
	instance "cloud.google.com/go/spanner/admin/instance/apiv1"
	"github.com/googleapis/gax-go/v2"
	"google.golang.org/api/option"
	databasepb "google.golang.org/genproto/googleapis/spanner/admin/database/v1"
	instancepb "google.golang.org/genproto/googleapis/spanner/admin/instance/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// nolint: gochecknoglobals
var (
	_port     = flag.Int("port", 9010, "endpoint")
	_endpoint = fmt.Sprintf("0.0.0.0:%d", *_port)
)

func main() {
	flag.Parse()

	log.Printf("Endpoint is set to: %s\n", _endpoint)

	ctx := context.Background()
	go func() {
		if err := ensureDatabase(ctx); err != nil {
			panic(err)
		}
	}()
	cmd := exec.Command("./gateway_main", "--hostname", "0.0.0.0")
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	cmd.Run()
}

func ensureDatabase(ctx context.Context) error {
	inst := os.Getenv("SPANNER_INSTANCE_ID")
	proj := os.Getenv("SPANNER_PROJECT_ID")
	db := os.Getenv("SPANNER_DATABASE_ID")

	if inst != "" && proj != "" {
		ic, err := instance.NewInstanceAdminClient(ctx,
			option.WithoutAuthentication(),
			option.WithGRPCDialOption(grpc.WithInsecure()),
			option.WithEndpoint(_endpoint),
		)
		if err != nil {
			return err
		}
		defer func() { _ = ic.Close() }()

		cir := &instancepb.CreateInstanceRequest{
			InstanceId: inst,
			Instance: &instancepb.Instance{
				Config:      "emulator-config",
				DisplayName: "",
				NodeCount:   1,
			},
			Parent: "projects/" + proj,
		}

		log.Printf("attempting to create instance %v\n", inst)
		if cirOp, err := ic.CreateInstance(ctx, cir, gax.WithGRPCOptions(grpc.WaitForReady(true))); err != nil {
			// get the status code
			if errStatus, ok := status.FromError(err); ok {
				// if the resource already exists, continue
				if errStatus.Code() == codes.AlreadyExists {
					log.Printf("instance already exists, continuing\n")
				} else {
					return err
				}
			} else {
				return err
			}
		} else {
			_, err = cirOp.Wait(ctx)
			if err != nil {
				return err
			}
			log.Println("instance created")
		}
	}

	if db != "" {
		dc, err := database.NewDatabaseAdminClient(ctx,
			option.WithoutAuthentication(),
			option.WithGRPCDialOption(grpc.WithInsecure()),
			option.WithEndpoint(_endpoint),
		)
		if err != nil {
			return err
		}
		defer func() { _ = dc.Close() }()
		log.Printf("attempting to create database %v\n", db)
		cdr := &databasepb.CreateDatabaseRequest{
			Parent:          "projects/" + proj + "/instances/" + inst,
			CreateStatement: "CREATE DATABASE `" + db + "`",
		}
		if cdrOp, err := dc.CreateDatabase(ctx, cdr); err != nil {
			// get the status code
			if errStatus, ok := status.FromError(err); ok {
				// if the resource already exists, continue
				if errStatus.Code() == codes.AlreadyExists {
					log.Printf("database already exists, continuing\n")
				} else {
					return err
				}
			} else {
				return err
			}
		} else {
			_, err = cdrOp.Wait(ctx)
			if err != nil {
				return err
			}
			log.Println("database created")
		}
	}

	return nil
}
