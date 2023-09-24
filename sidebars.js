/**
 * Creating a sidebar enables you to:
 - create an ordered group of docs
 - render a sidebar for each doc of that group
 - provide next/previous navigation
 The sidebars can be generated from the filesystem, or explicitly defined here.
 Create as many sidebars as you want.
 */

module.exports = {
  // By default, Docusaurus generates a sidebar from the docs folder structure
  documentationSidebar: [
    { 
    
    },
    'intro',
    {
      type: 'category',
      label: 'Getting Started',
      link: {
        type: 'generated-index',
        slug: '/getting-started',
        title: 'Getting Started',
        description: "Get Started with Bacalhau!",
      },
      collapsed: false,
      items: [
        'getting-started/architecture',
        'getting-started/installation',
        'getting-started/docker-workload-onboarding',
        'getting-started/wasm-workload-onboarding',
        'getting-started/resources'
      ],
    },
    {
      type: 'category',
      label: 'Examples',
      link: {
        type: 'generated-index',
        title: 'Examples',
        slug: '/examples',
        description: "Bacalhau comes pre-loaded with exciting examples to showcase its abilities and help get you started.",
      },
      collapsed: true,
      items: [
        {
          type: 'category',
          label: 'Case Studies',
          link: {
            type: 'generated-index',
            description: "Case Studies",  
          },
          items: [
            'examples/case-studies/duckdb-log-processing/index',
          ],
        },
        {
          type: 'category',
          label: 'Workload Onboarding',
          link: {
            type: 'generated-index',
            description: "This directory contains examples relating to performing common tasks with Bacalhau.",
          },
          items: [
            'examples/workload-onboarding/bacalhau-docker-image/index',
            'examples/workload-onboarding/Reading-From-Multiple-S3-Buckets/index',
            'examples/workload-onboarding/Running-Jupyter-Notebook/index',
            'examples/workload-onboarding/Prolog-Hello-World/index',
            'examples/workload-onboarding/Python-Custom-Container/index',
            'examples/workload-onboarding/python-pandas/index',
            'examples/workload-onboarding/r-custom-docker-prophet/index',
            'examples/workload-onboarding/r-hello-world/index',
            'examples/workload-onboarding/CUDA/index',
            'examples/workload-onboarding/rust-wasm/index',
            'examples/workload-onboarding/Sparkov-Data-Generation/index',
            'examples/workload-onboarding/custom-containers/index',
            'examples/workload-onboarding/CUDA/index',
            'examples/workload-onboarding/trivial-python/index',
            'examples/workload-onboarding/python-script/index',
          ],
        },
        {
          type: 'category',
          label: 'Data Engineering',
          link: {
            type: 'generated-index',
            description: "This directory contains examples relating to data engineering workloads. The goal is to provide a range of examples that show you how to work with Bacalhau in a variety of use cases.",  
          },
          items: [
            'examples/data-engineering/blockchain-etl/index',
            'examples/data-engineering/csv-to-avro-or-parquet/index',
            'examples/data-engineering/DuckDB/index',
            'examples/data-engineering/image-processing/index',
            'examples/data-engineering/oceanography-conversion/index',
            'examples/data-engineering/simple-parallel-workloads/index',
          ],
        },
        {
          type: 'category',
          label: 'Model Inference',
          link: {
            type: 'generated-index',
            description: "This directory contains examples relating to model inference workloads.",
          },
          items: [
            'examples/model-inference/Huggingface-Model-Inference/index',
            'examples/model-inference/object-detection-yolo5/index',
            'examples/model-inference/S3-Model-Inference/index',
            'examples/model-inference/Stable-Diffusion-CKPT-Inference/index',
            'examples/model-inference/stable-diffusion-cpu/index',
            'examples/model-inference/stable-diffusion-gpu/index',
            'examples/model-inference/StyleGAN3/index',
            'examples/model-inference/EasyOCR/index',
            'examples/model-inference/Openai-Whisper/index',
          ],
        },
        {
          type: 'category',
          label: 'Model Training',
          link: {
            type: 'generated-index',
            description: "This directory contains examples relating to model training workloads.",
          },
          items: [
            'examples/model-training/Stable-Diffusion-Dreambooth/index',
            'examples/model-training/Training-Pytorch-Model/index',
            'examples/model-training/Training-Tensorflow-Model/index',
          ],
        },
        {
          type: 'category',
          label: 'Molecular Dynamics',
          link: {
            type: 'generated-index',
            description: "This directory contains examples relating to performing common tasks with Bacalhau.",
          },
          items: [
            'examples/molecular-dynamics/BIDS/index',
            'examples/molecular-dynamics/Coreset/index',
            'examples/molecular-dynamics/Genomics/index',
            'examples/molecular-dynamics/Gromacs/index',
            'examples/molecular-dynamics/openmm/index',
          ],
        },
      ],
    },
    {
      type: 'category',
      label: 'Data Ingestion',
      link: {
        type: 'generated-index',
        slug: '/data-ingestion',
        title: 'Data Ingestion',
      },
      collapsed: true,
      items: [
        'data-ingestion/from-url',
        'data-ingestion/pin',
        'data-ingestion/s3',
      ],
    },
    {
      type: 'category',
      label: 'Process',
      link: {
        type: 'generated-index',
        slug: '/process',
        title: 'Process',
      },
      collapsed: true,
      items: [
        'next-steps/gpu',
        'next-steps/networking',
        'next-steps/private-cluster',
      ],
    },
    {
      type: 'category',
      label: 'Running a Node',
      link: {
        type: 'generated-index',
        title: 'Running a node',
        slug: '/running-node',
      },
      collapsed: true,
      items: [
        'running-node/quick-start',
        'running-node/quick-start-docker',
        'running-node/networking',
        'running-node/storage-providers',
        'running-node/job-selection',
        'running-node/resource-limits',
        'running-node/test-network',
        'running-node/gpu',
        'running-node/persistence',
        'running-node/configuring-tls',
        'running-node/windows-support',
        'running-node/observability'
      ],
    },
    {
      type: 'category',
      label: 'SDK',
      link: {
        type: 'generated-index',
        title: 'Running node',
        slug: '/sdk',
      },
      collapsed: true,
      items: [
        'sdk/python-sdk'
      ],
    },
    {
      type: 'category',
      label: 'FAQS',
      link: {
        type: 'generated-index',
        title: 'Troubleshooting',
        slug: '/troubleshooting',
      },
      collapsed: true,
      items: [
        'troubleshooting/debugging',
        'troubleshooting/faqs',   
        'all-flags',
      ],
    },
    {
      type: 'category',
      label: 'Integration',
      link: {
        type: 'generated-index',
        title: 'Integration',
        slug: '/integration',
      },
      collapsed: true,
      items: [
        'integration/apache-airflow',
        'integration/amplify',
        'integration/lilypad'
      ],
    },
    {
      type: 'category',
      label: 'Community',
      link: {
        type: 'generated-index',
        title: 'Community',
        slug: '/community',
      },
      collapsed: true,
      items: [
        'community/compute-landscape',
        'community/development',
        'community/style-guide',
        'community/ways-to-contribute',
      ],
    },
  ]
}
