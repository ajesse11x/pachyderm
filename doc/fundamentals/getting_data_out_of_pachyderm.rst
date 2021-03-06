.. _getting_data_out_of_pachyderm:


Getting Data Out of Pachyderm
=============================

Once you've got one or more pipelines built and have data flowing through Pachyderm,
you need to be able to track that data flowing through your pipeline(s) and 
get results out of Pachyderm. Let's use the `OpenCV pipeline <../getting_started/beginner_tutorial.html>`_ as an example. 

Here's what our pipeline and the corresponding data repositories look like:


.. image:: opencv.jpg
   :target: opencv.jpg
   :alt: alt tag

Every commit of new images into the "images" data repository results in a corresponding output commit of results into the "edges" data repository. But how do we get our results out of Pachyderm?  Moreover, how would we get the particular result corresponding to a particular input image?  That's what we will explore here.

Getting files with ``pachctl``
------------------------------

The ``pachctl`` CLI tool command `pachctl get file <../pachctl/pachctl_get_file.html>`_ can be used to get versioned data out of any data repository:

.. code-block:: sh

   pachctl get file <repo>@<branch-or-commit>:<path/to/file>

In the case of the OpenCV pipeline, we could get out an image named ``example_pic.jpg``:

.. code-block:: sh

   pachctl get file edges@master:example_pic.jpg

But how do we know which files to get?  Of course we can use the ``pachctl list file`` command to see what files are available.  But how do we know which results are the latest, came from certain input, etc.?  In this case, we would like to know which edge detected images in the ``edges`` repo come from which input images in the ``images`` repo.  This is where provenance and the ``flush commit`` command come in handy.

Examining file provenance with flush commit
-------------------------------------------

Generally, ``flush commit`` will let our process block on an input commit until all of the output results are ready to read. In other words, ``flush commit`` lets you view a consistent global snapshot of all your data at a given commit. Note, we are just going to cover a few aspects of ``flush commit`` here.

Let's demonstrate a typical workflow using ``flush commit``. First, we'll make a few commits of data into the ``images`` repo on the ``master`` branch.  That will then trigger our ``edges`` pipeline and generate three output commits in our ``edges`` repo:

.. code-block:: sh

   $ pachctl list commit images
   REPO                ID                                 PARENT                             STARTED              DURATION             SIZE
   images              c721c4bb9a8046f3a7319ed97d256bb9   a9678d2a439648c59636688945f3c6b5   About a minute ago   1 seconds            932.2 KiB
   images              a9678d2a439648c59636688945f3c6b5   87f5266ef44f4510a7c5e046d77984a6   About a minute ago   Less than a second   238.3 KiB
   images              87f5266ef44f4510a7c5e046d77984a6   <none>                             10 minutes ago       Less than a second   57.27 KiB
   $ pachctl list commit edges
   REPO                ID                                 PARENT                             STARTED              DURATION             SIZE
   edges               f716eabf95854be285c3ef23570bd836   026536b547a44a8daa2db9d25bf88b79   About a minute ago   Less than a second   233.7 KiB
   edges               026536b547a44a8daa2db9d25bf88b79   754542b89c1c47a5b657e60381c06c71   About a minute ago   Less than a second   133.6 KiB
   edges               754542b89c1c47a5b657e60381c06c71   <none>                             2 minutes ago        Less than a second   22.22 KiB

In this case, we have one output commit per input commit on ``images``.  However, this might get more complicated for pipelines with multiple branches, multiple PFS inputs, etc.  To confirm which commits correspond to which outputs, we can use ``flush commit``.  In particular, we can call ``flush commit`` on any one of our commits into ``images`` to see which output came from this particular commit:

.. code-block:: sh

   $ pachctl flush commit images@a9678d2a439648c59636688945f3c6b5
   REPO                ID                                 PARENT                             STARTED             DURATION             SIZE
   edges               026536b547a44a8daa2db9d25bf88b79   754542b89c1c47a5b657e60381c06c71   3 minutes ago       Less than a second   133.6 KiB

Exporting data by using ``egress``
--------------------------------------

In addition to getting data out of Pachyderm by using
``pachctl get file``\ , you can add an optional ``egress`` field
to your `pipeline specification <../reference/pipeline_spec.html>`_.
``egress`` enables you to push the results of a pipeline to an
external datastore such as Amazon S3, Google Cloud Storage, or
Azure Blob Storage. After the user code has finished running, but
before the job is marked as successful, Pachyderm pushes the data
to the specified destination.

You can specify the following ``egress`` protocols for the
corresponding storage:

.. note:: Use the horizontal scroll bar in the table below
   to view full descriptions and syntax.

.. list-table::
   :header-rows: 1

   * - Cloud Platform
     - Protocol
     - Description
     - Syntax
   * - Google Cloud Storage
     - ``gs://``
     - GCP uses the utility called ``gsutil`` to access GCP storage resources
       from a CLI. This utility uses the ``gs://`` prefix to access those
       resources.
     - ``gs://gs-bucket/gs-dir``
   * - Amazon S3
     - ``s3://``
     - The Amazon S3 storage protocol requires you to specify an ``s3://``
       prefix before the address of an Amazon resource. A valid address must
       include an endpoint and a bucket, and, optionally, a directory in
       your Amazon storage.
     - ``s3://s3-endpoint/s3-bucket/s3-dir``
   * - Azure Blob Storage
     - ``wasb://``
     - Microsoft Windows Azure Storage Blob (WASB) is the default Azure
       filesystem that outputs your data through HDInsight. To output your
       data to Azure Blob Storage, use the ``wasb://`` prefix, the container
       name, and your storage account in the  path to your directory.
     - ``wasb://default-container@storage-account/az-dir``


**Example:**

.. code-block:: bash

    "output_branch": string,
     "egress": {
       "URL": "s3://bucket/dir"
     },

Other ways to view, interact with, or export data in Pachyderm
--------------------------------------------------------------

Although ``pachctl`` and ``egress`` provide easy ways to interact with data in Pachyderm repos, they are by no means the only ways.  For example, you can:

* Have one or more of your pipeline stages connect and export data to databases running outside of Pachyderm.
* Use a Pachyderm service to launch a long running service, like Jupyter, that has access to internal Pachyderm data and can be accessed externally via a specified port.
* Mount versioned data from the distributed file system via ``pachctl mount ...`` (a feature best suited for experimentation and testing).
* If you're on Pachyderm Enterprise, you can use the s3gateway, which allows
  you to reuse existing tools or libraries that work with object stores.
  `See the s3gateway docs for more information <../enterprise/s3gateway.html>`_.
