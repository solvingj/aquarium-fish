#!/bin/sh -e

token=$1
[ "$token" ] || exit 1

label=xcode10.1
version=1

echo "Get or create the label"

label_id=$(curl -u "admin:$token" -k 'https://127.0.0.1:8001/api/v1/label/?filter=name="'$label'"%20AND%20version="'$version'"' | grep -o '"ID": *[0-9]\+,' | tr -dc '0-9')

if [ -z "$label_id" ]; then
    label_id=$(curl -u "admin:$token" -k -X POST -H 'Content-Type: application/json' -d '{"name":"'$label'", "version":'$version', "driver":"vmx",
        "definition": {
            "image": "macos1013-ci-xcode101",
            "images": {
                "macos1013": "https://artifact-storage/aquarium/image/macos1013-VERSION/macos1013-VERSION.tar.xz",
                "macos1013-ci": "https://artifact-storage/aquarium/image/macos1013-ci-VERSION/macos1013-ci-VERSION.tar.xz",
                "macos1013-ci-xcode101": "https://artifact-storage/aquarium/image/macos1013-ci-xcode101-VERSION/macos1013-ci-xcode101-VERSION.tar.xz"
            },
            "requirements": {
                "cpu": 14,
                "ram": 12,
                "disks": {
                    "xcode101": {
                        "type": "hfs+",
                        "size": 100,
                        "reuse": true
                    }
                }
            }
        },
        "metadata": {
            "JENKINS_AGENT_WORKSPACE": "/Volumes/xcode101"
        }
    }' https://127.0.0.1:8001/api/v1/label/ | grep -o '"ID": *[0-9]\+,' | tr -dc '0-9')
fi
echo "Label ID: ${label_id}"

echo "Press key to get label"
read w1

curl -u "admin:$token" -k https://127.0.0.1:8001/api/v1/label/${label_id}

echo "Press key to run application"
read w1

app_id=$(curl -u "admin:$token" -k -X POST -H 'Content-Type: application/json' -d '{"label_ID":'$label_id', "metadata":{
    "JENKINS_URL": "http://172.16.1.1:8085/",
    "JENKINS_AGENT_SECRET": "03839eabcf945b1e780be8f9488d264c4c57bf388546da9a84588345555f29b0",
    "JENKINS_AGENT_NAME": "test-node"
}}' https://127.0.0.1:8001/api/v1/application/ | grep -o '"ID": *[0-9]\+,' | tr -dc '0-9')
echo "Application ID: ${app_id}"

echo "Press key to check the application resource"
read w1

curl -u "admin:$token" -k https://127.0.0.1:8001/api/v1/application/$app_id/resource

echo "Press key to deallocate the application resource"
read w1

curl -u "admin:$token" -k https://127.0.0.1:8001/api/v1/application/$app_id/deallocate
