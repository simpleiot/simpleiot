module Pages.SignIn exposing (Model, Msg, page)

import Api.Auth exposing (User)
import Api.Data exposing (Data)
import Effect exposing (Effect)
import Element exposing (..)
import Element.Font as Font
import Element.Input as Input
import Gen.Params.SignIn exposing (Params)
import Page
import Request exposing (Request)
import Shared
import Storage exposing (Storage)
import UI.Form as Form
import UI.Style as Style
import View exposing (View)


page : Shared.Model -> Request.With Params -> Page.With Model Msg
page shared req =
    Page.advanced
        { init = init shared
        , update = update shared.storage
        , view = view
        , subscriptions = subscriptions
        }



-- INIT


type alias Model =
    { user : Data User
    , email : String
    , password : String
    , error : Maybe String
    }


init : Shared.Model -> ( Model, Effect Msg )
init shared =
    ( Model
        (case shared.storage.user of
            Just auth ->
                Api.Data.Success auth

            Nothing ->
                Api.Data.NotAsked
        )
        ""
        ""
        Nothing
    , Effect.none
    )



-- UPDATE


type Msg
    = EditEmail String
    | EditPass String
    | SignIn
    | GotUser (Data User)
    | NoOp


update : Storage -> Msg -> Model -> ( Model, Effect Msg )
update storage msg model =
    case msg of
        EditEmail email ->
            ( { model | email = String.toLower email }, Effect.none )

        EditPass password ->
            ( { model | password = password }, Effect.none )

        SignIn ->
            ( model
            , Effect.fromCmd <|
                Api.Auth.login
                    { user =
                        { email = model.email
                        , password = model.password
                        }
                    , onResponse = GotUser
                    }
            )

        NoOp ->
            ( model, Effect.none )

        GotUser user ->
            let
                error =
                    case user of
                        Api.Data.Success user_ ->
                            if user_.token /= "" then
                                Nothing

                            else
                                Just "Invalid login"

                        Api.Data.Failure _ ->
                            Just "Login Failure"

                        _ ->
                            Just "Login unknown state"
            in
            case Api.Data.toMaybe user of
                Just user_ ->
                    if user_.token /= "" then
                        ( { model | user = user, error = error }
                        , Effect.fromCmd <| Storage.signIn user_ storage
                        )

                    else
                        ( { model | user = user, error = error }, Effect.none )

                Nothing ->
                    ( { model | user = user, error = error }, Effect.none )



-- SUBSCRIPTIONS


subscriptions : Model -> Sub Msg
subscriptions model =
    Sub.none



-- VIEW


view : Model -> View Msg
view model =
    { title = "SIOT SignIn"
    , attributes = []
    , element =
        el [ centerX, centerY, Form.onEnter SignIn ] <|
            column
                [ spacing 32 ]
                [ viewError model.error
                , el [ Font.size 24, Font.semiBold ]
                    (text "Sign in")
                , column [ spacing 16 ]
                    [ Input.email
                        []
                        { onChange = \e -> EditEmail e
                        , text = model.email
                        , placeholder = Just <| Input.placeholder [] <| text "email"
                        , label = Input.labelAbove [] <| text "Email"
                        }
                    , Input.newPassword
                        []
                        { onChange = \p -> EditPass p
                        , show = False
                        , text = model.password
                        , placeholder = Just <| Input.placeholder [] <| text "password"
                        , label = Input.labelAbove [] <| text "Password"
                        }
                    , el [ alignRight ] <|
                        if String.isEmpty model.email then
                            Form.button
                                { label = "Sign In"
                                , color = Style.colors.gray
                                , onPress = NoOp
                                }

                        else
                            Form.button
                                { label = "Sign In"
                                , color = Style.colors.blue
                                , onPress = SignIn
                                }
                    ]
                ]
    }


viewError : Maybe String -> Element msg
viewError error =
    case error of
        Just err ->
            el Style.error (el [ centerX ] (text err))

        Nothing ->
            none
