/*
Copyright 2021.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controllers

import (
	"fmt"
	"time"

	"sigs.k8s.io/controller-runtime/pkg/client"

	v1alpha1 "github.com/open-cluster-management-io/cluster-backup-operator/api/v1alpha1"
	veleroapi "github.com/vmware-tanzu/velero/pkg/apis/velero/v1"
)

// returns then name of the last backup resource, or, if this is the first time to run the backup, a newly generated name
func getActiveBackupName(backup *v1alpha1.Backup, c client.Client) string {

	if backup.Status.CurrentBackup == "" {

		// no active backup, return newyly generated backup name
		return getVeleroBackupName(backup.Name, backup.Spec.VeleroConfig.Namespace)
	}

	if !isBackupFinished(backup.Status.VeleroBackup) {
		//if an active backup, return the CurrentBackup value
		return backup.Status.CurrentBackup
	}

	// no active backup, return newyly generated backup name
	return getVeleroBackupName(backup.Name, backup.Spec.VeleroConfig.Namespace)

}

func isBackupFinished(backup *veleroapi.Backup) bool {
	switch {
	case backup == nil:
		return false
	case backup.Status.Phase == "Completed" ||
		backup.Status.Phase == "Failed" ||
		backup.Status.Phase == "PartiallyFailed":
		return true
	}
	return false
}

// name used by the velero backup resource, created by the backup acm controller
func getVeleroBackupName(backupName, backupNamesapce string) string {
	return backupName + "-" + getFormattedTimeCRD(time.Now())
}

// Find takes a slice and looks for an element in it. If found it will
// return it's key, otherwise it will return -1 and a bool of false.
func Find(slice []string, val string) (int, bool) {
	for i, item := range slice {
		if item == val {
			return i, true
		}
	}
	return -1, false
}

//append unique value to a list
func appendUnique(slice []string, value string) []string {

	// check if the NS exists
	_, ok := Find(slice, value)
	if !ok {
		slice = append(slice, value)
	}
	return slice
}

// return current time formatted to validate k8s names
func getFormattedTimeCRD(t time.Time) string {
	formatted := fmt.Sprintf("%d-%02d-%02d-%02d%02d%02d",
		t.Year(), t.Month(), t.Day(),
		t.Hour(), t.Minute(), t.Second())
	return formatted
}

// return Duration in format 1h15m30s
func getFormattedDuration(duration time.Duration) string {

	formatted := duration.Truncate(time.Second).String()
	return formatted
}

// returns true if the interval required to wait for a backup has passed since the last backup execution
// or if there is no previous backup execution
func canStartBackup(backup *v1alpha1.Backup) bool {

	if backup.Status.VeleroBackup == nil {
		// no previous completed backup, can start one now
		return true
	}

	completedTime := backup.Status.VeleroBackup.Status.CompletionTimestamp.Time.Unix()
	if completedTime < 0 {
		// completion time not set, wait for it to be set
		return false
	}

	// interval in minutes, between backups
	interval := backup.Spec.Interval
	currentTime := time.Now().Unix()

	//can run another backup if current time - completed backup time is bigger then the interval in seconds
	return currentTime-completedTime >= int64(interval*60)

}