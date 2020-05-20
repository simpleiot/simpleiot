module Pages.SignIn exposing (Flags, Model, Msg, page)

import Element exposing (..)
import Element.Border as Border
import Element.Font as Font
import Element.Input as Input
import Global
import Html
import Html.Attributes as Attr
import Html.Events as Events
import Page exposing (Document, Page)
import UI.Styles as Styles


type alias Flags =
    ()


type alias Model =
    { email : String
    , password : String
    }


type Msg
    = UpdatedField Field String
    | ClickedSignIn


type Field
    = Email
    | Password


page : Page Flags Model Msg
page =
    Page.component
        { init = init
        , update = update
        , subscriptions = subscriptions
        , view = view
        }


init : Global.Model -> Flags -> ( Model, Cmd Msg, Cmd Global.Msg )
init _ _ =
    ( Model "" "", Cmd.none, Cmd.none )


update : Global.Model -> Msg -> Model -> ( Model, Cmd Msg, Cmd Global.Msg )
update _ msg model =
    case msg of
        UpdatedField Email value ->
            ( { model | email = value }
            , Cmd.none
            , Cmd.none
            )

        UpdatedField Password value ->
            ( { model | password = value }
            , Cmd.none
            , Cmd.none
            )

        ClickedSignIn ->
            ( model
            , Cmd.none
            , Global.send <|
                Global.SignIn model
            )


subscriptions : Global.Model -> Model -> Sub Msg
subscriptions _ _ =
    Sub.none


view : Global.Model -> Model -> Document Msg
view _ model =
    { title = "SignIn"
    , body =
        [ el [ centerX, centerY ] <|
            form
                { onSubmit = ClickedSignIn
                }
                [ spacing 32 ]
                [ el [ Font.size 24, Font.semiBold ]
                    (text "Sign in")
                , column [ spacing 16 ]
                    [ viewField
                        { label = "Email"
                        , onChange = UpdatedField Email
                        , inputType = EmailInput
                        , value = model.email
                        }
                    , viewField
                        { label = "Password"
                        , onChange = UpdatedField Password
                        , inputType = PasswordInput
                        , value = model.password
                        }
                    ]
                , el [ alignRight ] <|
                    if String.isEmpty model.email then
                        Input.button
                            (Styles.button Styles.colors.blue ++ [ alpha 0.6 ])
                            { onPress = Nothing
                            , label = text "Sign In"
                            }

                    else
                        Input.button
                            (Styles.button Styles.colors.blue ++ [ htmlAttribute (Attr.type_ "submit") ])
                            { onPress = Just ClickedSignIn
                            , label = text "Sign In"
                            }
                ]
        ]
    }


form : { onSubmit : msg } -> List (Attribute msg) -> List (Element msg) -> Element msg
form config attrs children =
    Element.html
        (Html.form
            [ Events.onSubmit config.onSubmit ]
            [ toHtml (column attrs children)
            ]
        )


toHtml : Element msg -> Html.Html msg
toHtml =
    Element.layoutWith { options = [ Element.noStaticStyleSheet ] } []


type InputType
    = EmailInput
    | PasswordInput


viewField :
    { inputType : InputType
    , label : String
    , onChange : String -> msg
    , value : String
    }
    -> Element msg
viewField config =
    let
        styles =
            { field =
                [ paddingXY 4 4
                , Border.rounded 0
                , Border.widthEach
                    { top = 0
                    , left = 0
                    , right = 0
                    , bottom = 1
                    }
                ]
            , label =
                [ Font.size 16
                , Font.semiBold
                ]
            }

        label =
            Input.labelAbove
                styles.label
                (text config.label)
    in
    case config.inputType of
        EmailInput ->
            Input.email styles.field
                { onChange = config.onChange
                , text = config.value
                , placeholder = Nothing
                , label = label
                }

        PasswordInput ->
            Input.currentPassword styles.field
                { onChange = config.onChange
                , text = config.value
                , placeholder = Nothing
                , label = label
                , show = False
                }
