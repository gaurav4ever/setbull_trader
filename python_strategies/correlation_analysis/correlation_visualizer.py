"""
Correlation Visualizer for the correlation analysis system.

This module handles the visualization of correlation matrices and networks.
"""

import pandas as pd
import numpy as np
import matplotlib.pyplot as plt
import seaborn as sns
import networkx as nx
import logging
from typing import Dict, Tuple, List
from pathlib import Path
from correlation_analysis.stock_clusterer import StockClusterer

logger = logging.getLogger(__name__)

class CorrelationVisualizer:
    """Visualize correlation matrices and networks."""
    
    def __init__(self, correlation_matrices: Dict[str, pd.DataFrame]):
        """
        Initialize the correlation visualizer.
        
        Args:
            correlation_matrices: Dictionary of correlation matrices with their types
        """
        self.correlation_matrices = correlation_matrices
        
    def create_heatmap(self, corr_type: str, figsize: Tuple[int, int] = (12, 10)) -> plt.Figure:
        """
        Create a heatmap of the correlation matrix.
        
        Args:
            corr_type: Type of correlation matrix to visualize
            figsize: Figure size (width, height)
            
        Returns:
            Matplotlib figure object
        """
        if corr_type not in self.correlation_matrices:
            raise ValueError(f"Correlation type {corr_type} not found")
            
        logger.info(f"Creating heatmap for {corr_type} correlation")
        
        # Create figure
        fig, ax = plt.subplots(figsize=figsize)
        
        # Create heatmap
        sns.heatmap(
            self.correlation_matrices[corr_type],
            annot=True,
            fmt='.2f',
            cmap='coolwarm',
            center=0,
            square=True,
            ax=ax,
            cbar_kws={'label': 'Correlation Coefficient'}
        )
        
        # Set title and labels
        ax.set_title(f'{corr_type.title()} Correlation Matrix', pad=20)
        ax.set_xlabel('Instruments', labelpad=10)
        ax.set_ylabel('Instruments', labelpad=10)
        
        # Rotate x-axis labels for better readability
        plt.xticks(rotation=45, ha='right')
        plt.yticks(rotation=0)
        
        # Adjust layout
        plt.tight_layout()
        
        return fig
        
    def create_network_graph(self, corr_matrix: pd.DataFrame, threshold: float = 0.5) -> nx.Graph:
        """
        Create a network graph of significant correlations.
        
        Args:
            corr_matrix: Correlation matrix
            threshold: Minimum correlation value to include in the graph
            
        Returns:
            NetworkX graph object
        """
        logger.info(f"Creating network graph with threshold {threshold}")
        
        # Create graph
        G = nx.Graph()
        
        # Add nodes
        for stock in corr_matrix.columns:
            G.add_node(stock)
            
        # Add edges for significant correlations
        for i, stock1 in enumerate(corr_matrix.columns):
            for j, stock2 in enumerate(corr_matrix.columns):
                if i < j:  # Avoid duplicate edges
                    corr = corr_matrix.iloc[i, j]
                    if abs(corr) > threshold:
                        G.add_edge(stock1, stock2, weight=abs(corr))
                        
        return G
        
    def visualize_clusters(self, clusterer: StockClusterer, original_df: pd.DataFrame, output_dir: str):
        """
        Visualize stock clusters and their performance.
        
        Args:
            clusterer: StockClusterer instance with clustering results
            original_df: DataFrame containing original P&L data
            output_dir: Directory to save visualizations
        """
        logger.info("Visualizing stock clusters")
        
        # Get cluster members
        cluster_members = clusterer.get_cluster_members()
        
        # Create cluster visualization
        fig, ax = plt.subplots(figsize=(12, 10))
        
        # Create a color map for clusters
        colors = plt.cm.tab10(np.linspace(0, 1, len(cluster_members)))
        
        # Create network graph
        G = self.create_network_graph(clusterer.correlation_matrix)
        
        # Get positions
        pos = nx.spring_layout(G, k=1, iterations=50)
        
        # Draw nodes with cluster colors
        for i, (cluster_id, stocks) in enumerate(cluster_members.items()):
            nx.draw_networkx_nodes(
                G, pos,
                nodelist=stocks,
                node_color=[colors[i]] * len(stocks),
                node_size=500,
                alpha=0.8,
                ax=ax
            )
            
        # Draw edges
        nx.draw_networkx_edges(
            G, pos,
            width=1.0,
            alpha=0.3,
            ax=ax
        )
        
        # Draw labels
        nx.draw_networkx_labels(G, pos, ax=ax)
        
        # Add legend
        legend_elements = [
            plt.Line2D([0], [0], marker='o', color='w', label=f'Cluster {i+1}',
                      markerfacecolor=colors[i], markersize=10)
            for i in range(len(cluster_members))
        ]
        ax.legend(handles=legend_elements, title='Clusters')
        
        # Set title
        ax.set_title('Stock Clusters Based on Correlation', pad=20)
        
        # Save figure
        output_path = Path(output_dir) / 'stock_clusters.png'
        fig.savefig(output_path, dpi=300, bbox_inches='tight')
        plt.close(fig)
        logger.info(f"Saved cluster visualization to {output_path}")
        
        # Create and save cluster performance table
        performance_df = clusterer.calculate_cluster_performance(original_df)
        performance_df.to_csv(Path(output_dir) / 'cluster_performance.csv', index=False)
        logger.info("Saved cluster performance metrics to CSV")
        
    def save_visualizations(self, output_dir: str):
        """
        Save all visualizations to the specified directory.
        
        Args:
            output_dir: Directory to save visualizations
        """
        logger.info(f"Saving visualizations to {output_dir}")
        
        # Create output directory if it doesn't exist
        Path(output_dir).mkdir(parents=True, exist_ok=True)
        
        # Save heatmaps
        for corr_type in self.correlation_matrices:
            fig = self.create_heatmap(corr_type)
            output_path = Path(output_dir) / f'{corr_type}_heatmap.png'
            fig.savefig(output_path, dpi=300, bbox_inches='tight')
            plt.close(fig)
            logger.info(f"Saved {corr_type} heatmap to {output_path}")
            
        # Save network graphs
        for corr_type, corr_matrix in self.correlation_matrices.items():
            G = self.create_network_graph(corr_matrix)
            
            # Skip if no significant correlations
            if G.number_of_edges() == 0:
                logger.info(f"No significant correlations found for {corr_type} network")
                continue
                
            # Create figure
            fig, ax = plt.subplots(figsize=(12, 10))
            
            # Draw network
            pos = nx.spring_layout(G, k=1, iterations=50)
            
            # Draw nodes
            nx.draw_networkx_nodes(
                G, pos, 
                ax=ax, 
                node_size=500, 
                node_color='lightblue',
                alpha=0.8
            )
            
            # Draw edges with weights
            edge_weights = [G[u][v]['weight'] for u, v in G.edges()]
            nx.draw_networkx_edges(
                G, pos, 
                ax=ax, 
                width=[w * 2 for w in edge_weights],  # Scale edge width by weight
                alpha=0.5,
                edge_color=edge_weights,
                edge_cmap=plt.cm.Blues
            )
            
            # Draw labels
            nx.draw_networkx_labels(G, pos, ax=ax)
            
            # Add colorbar for edge weights
            sm = plt.cm.ScalarMappable(cmap=plt.cm.Blues, norm=plt.Normalize(vmin=min(edge_weights), vmax=max(edge_weights)))
            plt.colorbar(sm, ax=ax, label='Correlation Strength')
            
            # Set title and labels
            ax.set_title(f'{corr_type.title()} Correlation Network', pad=20)
            
            # Save figure
            output_path = Path(output_dir) / f'{corr_type}_network.png'
            fig.savefig(output_path, dpi=300, bbox_inches='tight')
            plt.close(fig)
            logger.info(f"Saved {corr_type} network to {output_path}")
            
        logger.info("All visualizations saved successfully")
