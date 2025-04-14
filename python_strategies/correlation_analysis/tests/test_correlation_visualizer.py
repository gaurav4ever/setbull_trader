"""
Unit tests for the CorrelationVisualizer class.
"""

import pytest
import pandas as pd
import numpy as np
import matplotlib.pyplot as plt
from pathlib import Path
from correlation_analysis.correlation_visualizer import CorrelationVisualizer

@pytest.fixture
def sample_correlation_matrices():
    """Create sample correlation matrices for testing."""
    # Create sample data
    stocks = ['AAPL', 'MSFT', 'GOOGL', 'AMZN']
    data = {
        'spearman': pd.DataFrame(
            np.array([
                [1.0, 0.7, 0.3, -0.2],
                [0.7, 1.0, 0.5, 0.1],
                [0.3, 0.5, 1.0, 0.6],
                [-0.2, 0.1, 0.6, 1.0]
            ]),
            index=stocks,
            columns=stocks
        ),
        'binary': pd.DataFrame(
            np.array([
                [1.0, 0.8, 0.4, -0.3],
                [0.8, 1.0, 0.6, 0.2],
                [0.4, 0.6, 1.0, 0.7],
                [-0.3, 0.2, 0.7, 1.0]
            ]),
            index=stocks,
            columns=stocks
        )
    }
    return data

@pytest.fixture
def visualizer(sample_correlation_matrices):
    """Create a CorrelationVisualizer instance for testing."""
    return CorrelationVisualizer(sample_correlation_matrices)

def test_create_heatmap(visualizer):
    """Test creating a heatmap visualization."""
    # Test with valid correlation type
    fig = visualizer.create_heatmap('spearman')
    assert isinstance(fig, plt.Figure)
    plt.close(fig)
    
    # Test with invalid correlation type
    with pytest.raises(ValueError):
        visualizer.create_heatmap('invalid_type')

def test_create_network_graph(visualizer):
    """Test creating a network graph."""
    # Test with default threshold
    G = visualizer.create_network_graph(visualizer.correlation_matrices['spearman'])
    assert G.number_of_nodes() == 4  # Number of stocks
    assert G.number_of_edges() > 0
    
    # Test with higher threshold
    G = visualizer.create_network_graph(visualizer.correlation_matrices['spearman'], threshold=0.6)
    assert G.number_of_edges() > 0  # Should have some edges
    
    # Test with very high threshold
    G = visualizer.create_network_graph(visualizer.correlation_matrices['spearman'], threshold=0.9)
    assert G.number_of_edges() >= 0  # May or may not have edges

def test_save_visualizations(visualizer, tmp_path):
    """Test saving visualizations to a directory."""
    # Create temporary directory
    output_dir = tmp_path / "visualizations"
    
    # Save visualizations
    visualizer.save_visualizations(str(output_dir))
    
    # Check that files were created
    assert (output_dir / "spearman_heatmap.png").exists()
    assert (output_dir / "binary_heatmap.png").exists()
    assert (output_dir / "spearman_network.png").exists()
    assert (output_dir / "binary_network.png").exists()

def test_network_graph_edge_weights(visualizer):
    """Test that network graph edge weights are correct."""
    G = visualizer.create_network_graph(visualizer.correlation_matrices['spearman'])
    
    # Check edge weights
    for u, v, data in G.edges(data=True):
        expected_weight = abs(visualizer.correlation_matrices['spearman'].loc[u, v])
        assert abs(data['weight'] - expected_weight) < 1e-10

def test_heatmap_annotations(visualizer):
    """Test that heatmap annotations are correct."""
    fig = visualizer.create_heatmap('spearman')
    ax = fig.axes[0]
    
    # Check that annotations are present
    assert len(ax.texts) > 0
    
    # Check that values match the correlation matrix
    for text in ax.texts:
        x, y = text.get_position()
        value = float(text.get_text())
        expected_value = visualizer.correlation_matrices['spearman'].iloc[int(y), int(x)]
        assert abs(value - expected_value) < 1e-10
    
    plt.close(fig) 