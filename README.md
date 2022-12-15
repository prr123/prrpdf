# azulpdf
golang pdf library

The purpose of creating this library is building a library with an API:

+ reading pdf files into a data structure
+ use a pdf file as a template and fill-in spaces with golang (usage could be the creation of invoices, pay slips etc.)
+ fill-in pdf forms programmatically (tax forms, quaterly VAT filings,  other government forms)

Currently there is no effort to create the following:

+ merge pdf documents (there are plenty of programs including pdfcpu)
+ create a command line interface

If there is enough interest, the library could be integrated into a microservice.

## Libary Methods

### decodePdf
a method to read pdf files and fill the infoPdf data structure

### decodePdfToText
a method that creates a text file to analyse pdf files for human beings

### createPdf
a method to create pdf documents from text documents

to come:
### 

## current state
the library can decode a basic pdf form
currently working on parsing font objects
there is little documenation on embedding fonts, yet fonts define the corporate image of most users

15 December 2022

expect frequent updates
