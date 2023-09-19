FROM python:3.11

WORKDIR /app

COPY tests/ ./

RUN pip install -r requirements.txt

# Define the command to run pytest
CMD ["pytest"]