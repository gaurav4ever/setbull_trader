"""
Stock Clustering Module

This module handles clustering of stocks based on their correlation patterns.
"""

import pandas as pd
import numpy as np
from scipy.cluster.hierarchy import linkage, fcluster
from scipy.spatial.distance import squareform
import logging
from typing import Dict, List

logger = logging.getLogger(__name__)

class StockClusterer:
    """Cluster stocks based on correlation patterns."""
    
    def __init__(self, correlation_matrix: pd.DataFrame):
        """
        Initialize the stock clusterer.
        
        Args:
            correlation_matrix: DataFrame containing correlation values between stocks
        """
        self.correlation_matrix = correlation_matrix
        self.distance_matrix = None
        self.cluster_labels = None
        self.n_clusters = None
        
    def create_distance_matrix(self) -> pd.DataFrame:
        """Convert correlation matrix to distance matrix."""
        logger.info("Creating distance matrix from correlation matrix")
        
        self.correlation_matrix = self.correlation_matrix.dropna(axis=0, how='any')
        self.correlation_matrix = self.correlation_matrix.dropna(axis=1, how='any')

        # Get correlation values
        corr_values = self.correlation_matrix.values
        
        # Ensure the matrix is symmetric
        # Take the maximum of the upper and lower triangles to ensure positive semi-definite
        upper_tri = np.triu(corr_values)
        lower_tri = np.tril(corr_values)
        symmetric_corr = (corr_values + corr_values.T) / 2
        
        # Ensure diagonal is 1
        np.fill_diagonal(symmetric_corr, 1.0)
        
        # Convert to distance matrix using the formula: d = sqrt(2*(1-corr))
        # Clip correlation values to [-1, 1] to avoid numerical issues
        symmetric_corr = np.clip(symmetric_corr, -1.0, 1.0)
        distance_matrix = np.sqrt(2 * (1 - symmetric_corr))
        np.savetxt("distance_matrix_raw.txt", distance_matrix)
        
        # Ensure diagonal is zero
        np.fill_diagonal(distance_matrix, 0.0)

        # Ensure the matrix is symmetric
        distance_matrix = (distance_matrix + distance_matrix.T) / 2

        print("Correlation matrix index:")
        print(self.correlation_matrix.index)
        print("Correlation matrix columns:")
        print(self.correlation_matrix.columns)
        
        # Create DataFrame with proper index and columns
        self.distance_matrix = pd.DataFrame(
            distance_matrix,
            index=self.correlation_matrix.index,
            columns=self.correlation_matrix.columns
        )

        asymmetry = distance_matrix - distance_matrix.T
        print("Max asymmetry:", np.max(np.abs(asymmetry)))
        max_asymmetry = np.max(np.abs(asymmetry))
        logger.warning(f"Max asymmetry: {max_asymmetry}")

        assert np.allclose(self.distance_matrix, self.distance_matrix.T), "Symmetry check failed"
        assert np.allclose(np.diag(self.distance_matrix), 0), "Diagonal not zero"
        
        # Verify symmetry and non-negativity
        if not np.allclose(self.distance_matrix, self.distance_matrix.T, atol=1e-8):
            raise ValueError("Distance matrix is not symmetric")
        if not np.all(self.distance_matrix >= 0):
            raise ValueError("Distance matrix contains negative values")

            
        return self.distance_matrix
        
    def find_optimal_clusters(self, max_clusters: int = 10) -> int:
        """
        Determine optimal number of clusters using hierarchical clustering.
        
        Args:
            max_clusters: Maximum number of clusters to consider
            
        Returns:
            Optimal number of clusters
        """
        print("Correlation Matrix:")
        print(self.correlation_matrix)

        if self.distance_matrix is None:
            self.create_distance_matrix()
            
        logger.info(f"Finding optimal number of clusters (max: {max_clusters})")
        
        # Convert distance matrix to condensed form
        condensed_dist = squareform(self.distance_matrix.values)
        
        # Perform hierarchical clustering
        Z = linkage(condensed_dist, method='ward')
        
        # Calculate cluster quality metrics
        best_score = -np.inf
        best_n = 2
        
        for n in range(2, max_clusters + 1):
            labels = fcluster(Z, n, criterion='maxclust')
            # Calculate silhouette score
            score = self._calculate_silhouette_score(condensed_dist, labels)
            if score > best_score:
                best_score = score
                best_n = n
                
        logger.info(f"Optimal number of clusters: {best_n}")
        self.n_clusters = best_n
        return best_n
        
    def apply_hierarchical_clustering(self, n_clusters: int) -> np.ndarray:
        """
        Apply hierarchical clustering algorithm.
        
        Args:
            n_clusters: Number of clusters to create
            
        Returns:
            Array of cluster labels
        """
        if self.distance_matrix is None:
            self.create_distance_matrix()
            
        logger.info(f"Applying hierarchical clustering with {n_clusters} clusters")
        
        # Convert distance matrix to condensed form
        condensed_dist = squareform(self.distance_matrix.values)
        
        # Perform hierarchical clustering
        Z = linkage(condensed_dist, method='ward')
        self.cluster_labels = fcluster(Z, n_clusters, criterion='maxclust')
        
        return self.cluster_labels
        
    def get_cluster_members(self) -> Dict[int, List[str]]:
        """
        Get stocks in each cluster.
        
        Returns:
            Dictionary mapping cluster numbers to lists of stock names
        """
        if self.cluster_labels is None:
            raise ValueError("Clustering has not been performed yet")
            
        cluster_members = {}
        for i, label in enumerate(self.cluster_labels):
            stock = self.correlation_matrix.index[i]
            if label not in cluster_members:
                cluster_members[label] = []
            cluster_members[label].append(stock)
            
        return cluster_members
        
    def calculate_cluster_performance(self, original_df: pd.DataFrame) -> pd.DataFrame:
        """
        Calculate performance metrics by cluster.
        
        Args:
            original_df: DataFrame containing original P&L data
            
        Returns:
            DataFrame with performance metrics by cluster
        """
        if self.cluster_labels is None:
            raise ValueError("Clustering has not been performed yet")
            
        # Get cluster members
        cluster_members = self.get_cluster_members()
        
        # Calculate performance metrics for each cluster
        performance_data = []
        for cluster_id, stocks in cluster_members.items():
            # Get P&L data for stocks in this cluster
            cluster_data = original_df[stocks]
            
            # Calculate metrics
            total_pnl = cluster_data.sum().sum()
            avg_pnl = cluster_data.mean().mean()
            win_rate = (cluster_data > 0).mean().mean()
            std_pnl = cluster_data.std().mean()
            
            performance_data.append({
                'Cluster': cluster_id,
                'Number of Stocks': len(stocks),
                'Total P&L': total_pnl,
                'Average P&L': avg_pnl,
                'Win Rate': win_rate,
                'Standard Deviation': std_pnl,
                'Stocks': ', '.join(stocks)
            })
            
        return pd.DataFrame(performance_data)
        
    def _calculate_silhouette_score(self, condensed_dist: np.ndarray, labels: np.ndarray) -> float:
        """Calculate silhouette score for clustering quality."""
        n = len(labels)
        if n <= 1:
            return 0.0
            
        # Calculate a and b scores
        a = np.zeros(n)
        b = np.zeros(n)
        
        for i in range(n):
            # Calculate a(i)
            mask = labels == labels[i]
            if sum(mask) > 1:
                a[i] = np.mean(condensed_dist[i, mask])
            else:
                a[i] = 0
                
            # Calculate b(i)
            other_labels = set(labels) - {labels[i]}
            if other_labels:
                b[i] = min([
                    np.mean(condensed_dist[i, labels == l])
                    for l in other_labels
                ])
            else:
                b[i] = 0
                
        # Calculate silhouette score
        s = (b - a) / np.maximum(a, b)
        return np.mean(s) 