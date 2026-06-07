#!/bin/bash

if [ $# -lt 2 ]; then
    echo "usage: ./create-admin.sh <email> <password>"
    exit 1
fi

EMAIL=$1
PASSWORD=$2

echo "creating admin user with email '$EMAIL'..."

go run cmd/admin/main.go -email="$EMAIL" -password="$PASSWORD"

if [ $? -eq 0 ]; then
    echo "admin user created successfully"
else
    echo "failed to create admin user"
fi
