# Use an official Python runtime as the parent image
FROM python:3.11-slim

# Set the working directory in the container
WORKDIR /app

# Copy the current directory contents into the container at the working directory
COPY . .

# Install any needed packages specified in requirements.txt
RUN pip install --no-cache-dir -r requirements.txt

# Make port 8080 available to the world outside this container
EXPOSE 8080

# Define environment variable for record directory
ENV RECORD_DIRECTORY=/records

# Run main.py when the container launches
CMD [ "python", "./src/main.py" ]
