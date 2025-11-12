---
title: '5 Java Interview Tips and Tricks'
description: "I keep seeing these in interviews and take-home assessments. Here are 5 examples with code and output."
date: 2025-11-12
tags: ['java', 'programming', 'interview']
lang: 'en'
translation: ''
---


## 1. Constructor Chaining – It Always Starts from the Top
```java
class Animal {
    public Animal() {
        System.out.println("I am Animal");
    }
}

class Dog extends Animal {
    public Dog() {
        System.out.println("I am Dog");
    }
}

public class Main {
    public static void main(String[] args) {
        new Dog();
    }
}
```
Output:
```text
I am Animal
I am Dog
```
Java always calls the parent constructor first. Even if you don't write super(). If the parent has no default constructor, the code won't compile.


## 2. Variables vs Methods – They Follow Different Rules
```java
class Animal {
    int i = 0;
}

class Dog extends Animal {
    int i = 10;

    void bark() {
        System.out.println("Bark");
    }
}

public class Main {
    public static void main(String[] args) {
        Animal a = new Dog();
        a.bark();           
        System.out.println(a.i);  
    }
}
```
Output:
```text
Bark
0
```
Methods are resolved at runtime based on the object. Variables are resolved at compile time based on the reference type.


## 3. Private Methods Are NOT Overridden
```java
class Animal {
    private void sound() {
        System.out.println("Animal");
    }
}

class Dog extends Animal {
    public void sound() {
        System.out.println("Bark");
    }
}

public class Main {
    public static void main(String[] args) {
        Animal a = new Dog();
        ((Dog) a).sound(); 
    }
}
```
Dog.sound() is a completely separate method. Private methods do not participate in polymorphism.


## 4. Static Methods Are Hidden, Not Overridden
```java
class Animal {
    static void eat() {
        System.out.println("Animal eats");
    }
}

class Dog extends Animal {
    static void eat() {
        System.out.println("Dog eats");
    }
}

public class Main {
    public static void main(String[] args) {
        Animal a = new Dog();
        a.eat(); 
    }
}
```
Static methods belong to the class. No polymorphism — resolved by reference type.


## 5. Instance Initializer Block (IIB) – Runs BEFORE the Constructor
```java
class Animal {
    { System.out.println("1. Animal IIB"); }
    Animal() { System.out.println("2. Animal Constructor"); }
    static { System.out.println("Animal Static"); }
}

class Dog extends Animal {
    { System.out.println("3. Dog IIB"); }
    Dog() { System.out.println("4. Dog Constructor"); }
    static { System.out.println("Dog Static"); }
}

public class Main {
    public static void main(String[] args) {
        new Dog();
    }
}
```
Output:
```text
Animal Static
Dog Static
1. Animal IIB
2. Animal Constructor
3. Dog IIB
4. Dog Constructor
```
Order:

Static blocks (parent → child)

IIB + parent constructor

Your constructor