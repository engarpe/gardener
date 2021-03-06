// Copyright (c) 2019 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and

package framework

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/mholt/archiver"

	"k8s.io/helm/pkg/helm/environment"

	"k8s.io/helm/pkg/downloader"
	"k8s.io/helm/pkg/getter"
	"k8s.io/helm/pkg/helm/helmpath"
	"k8s.io/helm/pkg/repo"
)

const (
	stableRepository = "stable"
)

// downloadChart downloads a native chart with <name> to <downloadDestination> from <stableRepoURL>
func downloadChart(ctx context.Context, name, downloadDestination, stableRepoURL string, helmSettings HelmAccess) (string, error) {
	providers := getter.All(environment.EnvSettings{})
	dl := downloader.ChartDownloader{
		Getters:  providers,
		HelmHome: helmpath.Home(helmSettings.HelmPath),
		Out:      os.Stdout,
	}

	err := ensureCacheIndex(ctx, helmSettings, stableRepoURL, providers)
	if err != nil {
		return "", err
	}

	// Download the chart
	filename, _, err := dl.DownloadTo(name, "", downloadDestination)
	if err != nil {
		return "", err
	}

	lname, err := filepath.Abs(filename)
	if err != nil {
		return "", err
	}

	err = archiver.Unarchive(lname, downloadDestination)
	if err != nil {
		return "", err
	}

	err = os.Remove(lname)
	if err != nil {
		return "", err
	}
	return lname, nil
}

func ensureCacheIndex(ctx context.Context, helmSettings HelmAccess, stableRepoURL string, providers getter.Providers) error {
	// This will download the cache index file only if it does not exist
	stableRepoCacheIndexPath := helmSettings.HelmPath.CacheIndex(stableRepository)
	if _, err := os.Stat(stableRepoCacheIndexPath); err != nil {
		if os.IsNotExist(err) {
			directory := filepath.Dir(stableRepoCacheIndexPath)
			err := os.MkdirAll(directory, os.ModePerm)
			if err != nil {
				return err
			}
			_, err = downloadCacheIndex(ctx, stableRepoCacheIndexPath, stableRepoURL, providers)
			if err != nil {
				return err
			}
		}
		return err
	}
	return nil
}

// downloadCacheIndex downloads the cache index for repository
func downloadCacheIndex(ctx context.Context, cacheFile, stableRepositoryURL string, providers getter.Providers) (*repo.Entry, error) {
	c := repo.Entry{
		Name:  stableRepository,
		URL:   stableRepositoryURL,
		Cache: cacheFile,
	}

	r, err := repo.NewChartRepository(&c, providers)
	if err != nil {
		return nil, err
	}

	if err := r.DownloadIndexFile(""); err != nil {
		return nil, fmt.Errorf("looks like %q is not a valid chart repository or cannot be reached: %s", stableRepositoryURL, err.Error())
	}

	return &c, nil
}
