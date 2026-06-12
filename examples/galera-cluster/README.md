# Galera Cluster Example

This example demonstrates how to create a Galera Cluster service using the SkySQL Terraform provider.

## Galera Cluster Requirements

- **Tier**: PowerPlus (mandatory)
- **Nodes**: 3 or 5 (odd numbers for quorum)
- **MaxScale Nodes**: 1 or 2
- **Minimum Size**: sky-4x16 (sky-2x8 is disabled for Galera)
- **Service Type**: transactional
- **Topology**: galera

## Usage

1. Set your SkySQL API key:
   ```bash
   export TF_SKYSQL_API_KEY="your-api-key-here"
   ```

2. Initialize Terraform:
   ```bash
   terraform init
   ```

3. Review the plan:
   ```bash
   terraform plan -var="project_id=your-project-id"
   ```

4. Apply the configuration:
   ```bash
   terraform apply -var="project_id=your-project-id"
   ```

## Galera Cluster Characteristics

- **Multi-master replication**: All nodes can accept writes
- **Synchronous replication**: Ensures data consistency across all nodes
- **Automatic failover**: Built-in high availability
- **Read/Write ports**: MaxScale provides both readwrite (3306) and readonly (3307) ports
- **Quorum-based**: Requires odd number of nodes for split-brain protection

## Scaling

To scale your Galera cluster:
- You can increase nodes from 3 to 5
- You can scale MaxScale nodes from 1 to 2
- You can upgrade instance size (e.g., sky-4x16 to sky-8x32)
- You can increase storage capacity

Note: Decreasing nodes is not recommended for production systems.
