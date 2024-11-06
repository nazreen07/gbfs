Fun with GBFS
- GBFS is a simple standard for publishing bike sharing feeds. https://github.com/MobilityData/gbfs/blob/master/gbfs.md
### Requirements
- Choose 3 providers of GBFS from this list https://github.com/MobilityData/gbfs/blob/master/systems.csv
- Design and deploy a solution that monitors changes in JSON files published by the providers and pull out stats about number of vehicles to display it in a dashboard with a historical overview.
- You have the freedom to decide on the stats and the dashboard design.
- CI/CD pipeline.
- Infrastructure needed must be defined as code.

### Bonus Points
- Make providers configurable.
- Deployed version on a cloud provider.
- Advanced comparisons between providers.
- Include alerts.
 
#Implementation

###code 
- It contains the application source code and Dockerfile
- The simple go code will get teh data from teh providerurl and ingest every 1 minute

###config
- It contains the manifest files to be applied to the cluster

###templates
- It contains the template.json and parameter.json to create the AKS cluster

###Pipelines
- aks-pipeline.yaml -> To create the AKS cluster
- role-assignment-pipeline.yaml -> To apply the role assignement to give acr access to the cluster
- build-deploy-pipeline -> Build and deploy the application into the namespace in the cluster
